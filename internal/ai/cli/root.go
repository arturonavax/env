package cli

import (
	"context"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/ai/daemon"
	"github.com/arturonavax/env/internal/ai/gitflow"
	"github.com/arturonavax/env/internal/ai/harness"
	"github.com/arturonavax/env/internal/ai/pipeline"
	"github.com/arturonavax/env/internal/ai/quota"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

const DefaultVersion = "1.0.0"

// OptimizeParams holds parsed CLI parameters for the system hardware & security optimizer.
type OptimizeParams struct {
	DryRun       bool
	Apply        bool
	AuditOnly    bool
	SkipLynis    bool
	Wizard       bool
	OutputFile   string
	Backend      string
	Model        string
	ValidateFile string
	Out          io.Writer
	ErrOut       io.Writer
}

// OptimizeFunc is an optional external hook for the optimize/sysopt subcommand.
type OptimizeFunc func(ctx context.Context, params OptimizeParams) error

type options struct {
	use            string
	version        string
	configPath     string
	stateDir       string
	workDir        string
	plat           *platform.Info
	runner         *security.Runner
	harnessRunner  harness.ProcessRunner
	lookPath       func(string) (string, error)
	connectivity   *daemon.ConnectivityOptions
	optimizeFunc   OptimizeFunc
	hasExplicitTTY bool
	explicitTTY    bool
}

// Option customizes the constructed aitools / anx ai Cobra command tree.
type Option func(*options)

// WithUse sets the root command Use token (e.g. "aitools" standalone or "ai" when mounted under anx).
func WithUse(use string) Option {
	return func(o *options) {
		if use != "" {
			o.use = use
		}
	}
}

// WithVersion sets the embedded CLI version string.
func WithVersion(v string) Option {
	return func(o *options) {
		if v != "" {
			o.version = v
		}
	}
}

// WithConfigPath sets the default config.json path (overridable via --config).
func WithConfigPath(p string) Option {
	return func(o *options) { o.configPath = p }
}

// WithStateDir sets the directory for state.json and daemon.log.
func WithStateDir(d string) Option {
	return func(o *options) { o.stateDir = d }
}

// WithWorkDir overrides the working directory for git and pipeline commands.
func WithWorkDir(d string) Option {
	return func(o *options) { o.workDir = d }
}

// WithPlatform injects a custom platform.Info instance.
func WithPlatform(p *platform.Info) Option {
	return func(o *options) { o.plat = p }
}

// WithRunner injects a custom security.Runner instance.
func WithRunner(r *security.Runner) Option {
	return func(o *options) { o.runner = r }
}

// WithHarnessRunner injects a custom harness.ProcessRunner instance.
func WithHarnessRunner(r harness.ProcessRunner) Option {
	return func(o *options) { o.harnessRunner = r }
}

// WithLookPath overrides executable PATH resolution for testing.
func WithLookPath(fn func(string) (string, error)) Option {
	return func(o *options) { o.lookPath = fn }
}

// WithConnectivity overrides daemon TCP/DNS connectivity check options.
func WithConnectivity(c daemon.ConnectivityOptions) Option {
	return func(o *options) { o.connectivity = &c }
}

// WithOptimizeFunc registers an optional custom handler for the optimize/sysopt subcommand.
func WithOptimizeFunc(fn OptimizeFunc) Option {
	return func(o *options) { o.optimizeFunc = fn }
}

// WithIsTTY forces TTY detection to a deterministic boolean for tests.
func WithIsTTY(isTTY bool) Option {
	return func(o *options) {
		o.hasExplicitTTY = true
		o.explicitTTY = isTTY
	}
}

type runtimeEnv struct {
	opts       options
	cfgFlag    string
	configPath string
	stateDir   string
	cfg        *config.Config
	store      *config.StateStore
	plat       *platform.Info
	runner     *security.Runner
	registry   *harness.Registry
	quotaSvc   *quota.Service
	daemonMgr  *daemon.Manager
	gitSvc     *gitflow.Service
	pipeRunner *pipeline.Runner
}

func (rt *runtimeEnv) init() error {
	if rt.plat == nil {
		if rt.opts.plat != nil {
			rt.plat = rt.opts.plat
		} else {
			rt.plat = platform.Detect()
		}
	}
	if rt.runner == nil {
		if rt.opts.runner != nil {
			rt.runner = rt.opts.runner
		} else {
			rt.runner = security.NewRunner(60 * time.Second)
		}
	}

	cfgPath := rt.cfgFlag
	if cfgPath == "" {
		cfgPath = rt.opts.configPath
	}
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath(rt.plat.HomeDir())
	}
	rt.configPath = cfgPath

	stateDir := rt.opts.stateDir
	if stateDir == "" {
		stateDir = config.DefaultStateDir(rt.plat.HomeDir())
	}
	rt.stateDir = stateDir

	cfg, err := config.Load(rt.configPath)
	if err != nil {
		// Fall back safely to defaults while warning only if needed; explicit syntax errors surface cleanly.
		return err
	}
	rt.cfg = cfg
	rt.store = config.NewStateStore(rt.stateDir)

	lookPath := rt.opts.lookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	regOpts := []harness.RegistryOption{harness.WithLookPath(lookPath)}
	if rt.opts.harnessRunner != nil {
		regOpts = append(regOpts, harness.WithProcessRunner(rt.opts.harnessRunner))
	}
	rt.registry = harness.NewRegistry(rt.cfg, regOpts...)

	connOpts := daemon.DefaultConnectivityOptions(45 * time.Second)
	if rt.opts.connectivity != nil {
		connOpts = *rt.opts.connectivity
	}

	rt.quotaSvc = quota.NewService(
		rt.cfg,
		rt.store,
		quota.WithExecutor(rt.runner),
		quota.WithNetworkChecker(func(ctx context.Context) error {
			return daemon.WaitForConnectivityWithOptions(ctx, connOpts)
		}),
	)

	rt.daemonMgr = daemon.NewManager(
		rt.cfg,
		rt.store,
		rt.quotaSvc,
		daemon.WithConfigPath(rt.configPath),
		daemon.WithPlatform(rt.plat),
		daemon.WithCommandExecutor(rt.runner),
		daemon.WithLookPath(lookPath),
		daemon.WithConnectivity(connOpts),
	)

	rt.gitSvc = gitflow.NewService(rt.cfg, rt.registry, rt.runner)
	rt.pipeRunner = pipeline.NewRunner(rt.cfg, rt.registry, rt.runner)
	return nil
}

func (rt *runtimeEnv) isInteractive(cmd *cobra.Command) bool {
	if rt.opts.hasExplicitTTY {
		return rt.opts.explicitTTY
	}
	inFile, inOK := cmd.InOrStdin().(*os.File)
	outFile, outOK := cmd.OutOrStdout().(*os.File)
	if !inOK || !outOK {
		return false
	}
	return term.IsTerminal(inFile.Fd()) && term.IsTerminal(outFile.Fd())
}

func (rt *runtimeEnv) isStdinTTY(cmd *cobra.Command) bool {
	if rt.opts.hasExplicitTTY {
		return rt.opts.explicitTTY
	}
	inFile, ok := cmd.InOrStdin().(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(inFile.Fd())
}

// NewCommand constructs a fresh, isolated Cobra command tree suitable for standalone `aitools`
// or mounted as `anx ai`.
func NewCommand(opts ...Option) *cobra.Command {
	o := options{
		use:     "aitools",
		version: DefaultVersion,
	}
	for _, opt := range opts {
		opt(&o)
	}

	rt := &runtimeEnv{opts: o}

	root := &cobra.Command{
		Use:           o.use,
		Short:         "AI productivity, rolling quota management, and workflow automation suite",
		Long:          "Multi-harness AI CLI suite providing rolling quota inspection, hardware RTC auto-wake daemon, conventional git commits, PR generation, system optimization, and declarative AI pipelines.",
		Version:       o.version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Chain any ancestor PersistentPreRunE when mounted as a subcommand (e.g. `anx ai`).
			for p := cmd.Parent(); p != nil; p = p.Parent() {
				if p != cmd && p.PersistentPreRunE != nil && p.Name() != o.use {
					if err := p.PersistentPreRunE(cmd, args); err != nil {
						return err
					}
					break
				}
			}
			return rt.init()
		},
	}

	root.PersistentFlags().StringVar(&rt.cfgFlag, "config", "", "Path to aitools JSON/JSONC configuration file")

	root.AddGroup(
		&cobra.Group{ID: "workflows", Title: "AI Workflows & Pipelines:"},
		&cobra.Group{ID: "quota", Title: "Quota & Background Daemon:"},
		&cobra.Group{ID: "system", Title: "Configuration & Diagnostics:"},
	)

	root.AddCommand(
		newLLMCmd(rt),
		newCommitCmd(rt),
		newPRCmd(rt),
		newRunCmd(rt),
		newCommandsCmd(rt),
		newOptimizeCmd(rt),
		newQuotaCmd(rt),
		newWatchCmd(rt),
		newWakeCmd(rt),
		newDaemonCmd(rt),
		newStatusCmd(rt),
		newBackendsCmd(rt),
		newConfigCmd(rt),
		newSetupCmd(rt),
	)

	return root
}
