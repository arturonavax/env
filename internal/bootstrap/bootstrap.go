package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/arturonavax/env/internal/app"
	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/hacks"
	"github.com/arturonavax/env/internal/journal"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/providers"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/snapshot"
	"github.com/arturonavax/env/internal/state"
	"github.com/arturonavax/env/internal/tools"
)

// Container holds the assembled composition root services and infrastructure singletons.
type Container struct {
	Platform     *platform.Info
	Catalog      *catalog.Registry
	Journal      *journal.FileJournal
	Snapshots    *snapshot.Manager
	Store        *state.Store
	Runner       *security.Runner
	Provider     providers.Provider
	InstallSvc   *app.InstallService
	SyncSvc      *app.SyncService
	DoctorSvc    *app.DoctorService
	AuditSvc     *app.AuditService
	HacksSvc     *app.HacksService
	OpsSvc       *app.OpsService
	ToolsSvc     *app.ToolsService
	ToolRunner   *tools.Runner
	ControlPlane *app.ControlPlaneService
	DestroyerSvc *app.DestroyerService
	SecuritySeal *security.OriginSeal
	RepoRoot     string
	Viper        *viper.Viper
	Version      string
	Commit       string
	Date         string
}

// Config specifies runtime options for constructing a Container.
type Config struct {
	RepoRoot      string
	Version       string
	Commit        string
	Date          string
	Offline       bool
	TrustVerifier security.DirectoryTrustVerifier
}

// Option modifies Config.
type Option func(*Config)

// WithTrustVerifier allows injecting a custom directory trust verifier.
func WithTrustVerifier(v security.DirectoryTrustVerifier) Option {
	return func(c *Config) {
		c.TrustVerifier = v
	}
}

// Close closes runtime resources held by the container, including state storage and journals.
func (c *Container) Close() error {
	if c == nil {
		return nil
	}
	var errs []error
	if c.Store != nil {
		if err := c.Store.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// WithRepoRoot sets an explicit repository root override.
func WithRepoRoot(repoRoot string) Option {
	return func(c *Config) {
		c.RepoRoot = repoRoot
	}
}

// WithVersion sets build metadata.
func WithVersion(version, commit, date string) Option {
	return func(c *Config) {
		c.Version = version
		c.Commit = commit
		c.Date = date
	}
}

// WithOffline sets offline network restriction.
func WithOffline(offline bool) Option {
	return func(c *Config) {
		c.Offline = offline
	}
}

// NewLazyContainer constructs a lightweight Container without touching the filesystem,
// opening state databases, initializing journals, or creating snapshot directories.
// This is used for commands like version, completion, and help queries.
func NewLazyContainer(opts ...Option) (*Container, error) {
	cfg := Config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	plat := platform.Detect()
	v := viper.New()
	v.SetEnvPrefix("ANX")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	cat := catalog.NewDefaultRegistry()
	sealOpts := []security.SealOption{
		security.WithSealRepoRoot(cfg.RepoRoot),
		security.WithSealSkipDotfiles(true),
	}
	if cfg.TrustVerifier != nil {
		sealOpts = append(sealOpts, security.WithSealTrustVerifier(cfg.TrustVerifier))
	}
	seal, _ := security.CreateOriginSeal(sealOpts...)

	return &Container{
		Platform:     plat,
		Catalog:      cat,
		SecuritySeal: seal,
		Viper:        v,
		Version:      cfg.Version,
		Commit:       cfg.Commit,
		Date:         cfg.Date,
	}, nil
}

// NewContainer initializes and wires the production dependency graph.
func NewContainer(opts ...Option) (*Container, error) {
	cfg := Config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	plat := platform.Detect()

	v := viper.New()
	v.SetEnvPrefix("ANX")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(plat.ConfigDir())
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("reading %s/config.yaml: %w", plat.ConfigDir(), err)
		}
	}

	store, err := state.OpenStore(plat.StateDir())
	if err != nil {
		return nil, err
	}
	var initSuccess bool
	defer func() {
		if !initSuccess && store != nil {
			_ = store.Close()
		}
	}()

	repoRoot := cfg.RepoRoot
	if repoRoot == "" {
		repoRoot = v.GetString("repo")
	}
	repoRoot = ResolveRepoRoot(repoRoot, plat.HomeDir(), store.RepoRoot())
	if isDir(filepath.Join(repoRoot, "files")) && isDir(filepath.Join(repoRoot, "manifests")) {
		_ = store.SetRepoRoot(repoRoot)
	}

	cat, err := catalog.NewRegistryWithOverlays(
		filepath.Join(repoRoot, "manifests"),
		filepath.Join(plat.ConfigDir(), "manifests.d"),
	)
	if err != nil {
		return nil, err
	}

	j, err := journal.NewFileJournal(plat.StateDir())
	if err != nil {
		return nil, err
	}
	snap, err := snapshot.NewManager(plat.StateDir())
	if err != nil {
		return nil, err
	}

	runner := security.NewRunner(60 * time.Second)
	installRunner := security.NewRunner(30 * time.Minute)
	prov := providers.NewDispatcher(plat, cat.Catalog(), installRunner, repoRoot)

	syncSvc := app.NewSyncService(plat, cat, snap, store, repoRoot)
	installSvc := app.NewInstallService(app.InstallDeps{
		Platform: plat, Catalog: cat, Journal: j, Snapshots: snap, Provider: prov, Store: store, Sync: syncSvc,
	})

	osvClient := security.NewOSVClient(
		10*time.Second,
		security.WithOSVCacheDir(filepath.Join(plat.CacheDir(), "osv"), 6*time.Hour),
	)

	repoHacksDir := filepath.Join(repoRoot, "manifests", "hacks")
	if _, err := os.Stat(repoHacksDir); err != nil {
		repoHacksDir = filepath.Join(repoRoot, "hacks")
	}

	hackReg, err := hacks.LoadRegistry(
		hacks.WithRepoHacksDir(repoHacksDir),
		hacks.WithUserHacksDir(filepath.Join(plat.ConfigDir(), "hacks.d")),
	)
	if err != nil {
		return nil, err
	}

	toolReg, err := tools.LoadRegistry(
		tools.WithRepoToolsDir(filepath.Join(repoRoot, "manifests")),
		tools.WithUserToolsDir(filepath.Join(plat.ConfigDir(), "tools.d")),
		tools.WithPlatform(plat),
	)
	if err != nil {
		return nil, err
	}
	toolRunner := tools.NewRunner(toolReg)
	toolUpdater := tools.NewUpdater(toolReg, func(ctx context.Context, packageID string) error {
		plan, err := installSvc.Plan(ctx, "all", []string{packageID})
		if err != nil {
			return err
		}
		return installSvc.Apply(ctx, plan)
	})
	toolsSvc := app.NewToolsService(toolReg, toolRunner, toolUpdater)
	doctorSvc := app.NewDoctorService(plat, cat, app.WithDoctorStore(store), app.WithDoctorRepoRoot(repoRoot))
	auditSvc := app.NewAuditService(cat, prov, osvClient)
	hacksSvc := app.NewHacksService(plat, hackReg, installRunner, snap, store, repoRoot)
	opsSvc := app.NewOpsService(plat, installRunner)
	controlPlane := app.NewControlPlaneService(plat, cat, installSvc, syncSvc, hacksSvc, toolsSvc, store)
	destroyerSvc := app.NewDestroyerService(plat, cat, installSvc, syncSvc, store, app.WithDestroyerToolsRegistry(toolReg))
	sealOpts := []security.SealOption{
		security.WithSealRepoRoot(repoRoot),
		security.WithSealUserDropIn(filepath.Join(plat.ConfigDir(), "manifests.d")),
		security.WithSealTargetHome(plat.HomeDir()),
	}
	if cfg.TrustVerifier != nil {
		sealOpts = append(sealOpts, security.WithSealTrustVerifier(cfg.TrustVerifier))
	}
	seal, _ := security.CreateOriginSeal(sealOpts...)

	initSuccess = true

	return &Container{
		Platform:     plat,
		Catalog:      cat,
		Journal:      j,
		Snapshots:    snap,
		Store:        store,
		Runner:       runner,
		Provider:     prov,
		InstallSvc:   installSvc,
		SyncSvc:      syncSvc,
		DoctorSvc:    doctorSvc,
		AuditSvc:     auditSvc,
		HacksSvc:     hacksSvc,
		OpsSvc:       opsSvc,
		ToolsSvc:     toolsSvc,
		ToolRunner:   toolRunner,
		ControlPlane: controlPlane,
		DestroyerSvc: destroyerSvc,
		SecuritySeal: seal,
		RepoRoot:     repoRoot,
		Viper:        v,
		Version:      cfg.Version,
		Commit:       cfg.Commit,
		Date:         cfg.Date,
	}, nil
}

// ResolveRepoRoot finds the env repository: configured path, env vars, walking up from cwd, persisted store root, or well-known user paths.
func ResolveRepoRoot(configured, home, stored string) string {
	if configured != "" && isDir(filepath.Join(configured, "manifests")) {
		return configured
	}
	if envRepo := os.Getenv("ANX_REPO"); envRepo != "" && isDir(filepath.Join(envRepo, "manifests")) {
		return envRepo
	}
	if envRepoRoot := os.Getenv("REPO_ROOT"); envRepoRoot != "" && isDir(filepath.Join(envRepoRoot, "manifests")) {
		return envRepoRoot
	}
	cwd, err := os.Getwd()
	if err == nil {
		for dir := cwd; ; dir = filepath.Dir(dir) {
			if isDir(filepath.Join(dir, "files")) && isDir(filepath.Join(dir, "manifests")) {
				return dir
			}
			if dir == filepath.Dir(dir) {
				break
			}
		}
	}
	if stored != "" && isDir(filepath.Join(stored, "files")) && isDir(filepath.Join(stored, "manifests")) {
		return stored
	}
	candidates := []string{
		"/repo",
		filepath.Join(home, "repos", "github.com", "arturonavax", "env"),
		filepath.Join(home, ".local", "share", "anx", "repo"),
		filepath.Join(home, "env"),
		filepath.Join(home, ".dotfiles"),
	}
	for _, cand := range candidates {
		if isDir(filepath.Join(cand, "files")) && isDir(filepath.Join(cand, "manifests")) {
			return cand
		}
	}
	if cwd != "" {
		return cwd
	}
	return "."
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
