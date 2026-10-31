package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/catalog"
	"github.com/arturonavax/env/internal/configsync"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/snapshot"
	"github.com/arturonavax/env/internal/state"
)

// SyncOptions tunes a synchronization run.
type SyncOptions struct {
	Group    string
	DryRun   bool
	Force    bool   // incoming (repo) wins on conflicts
	Push     bool   // capture from $HOME back into the repository instead
	Shell    string // override shell for filtering configs (defaults to platform shell)
	Terminal string // override terminal for filtering configs (defaults to platform terminal)
}

// SyncResult reports what happened for one configuration.
type SyncResult struct {
	ConfigID      string   `json:"config_id"`
	Target        string   `json:"target"`
	Action        string   `json:"action"` // created|updated|merged|unchanged|conflict|captured|skipped|drift
	Files         int      `json:"files,omitempty"`
	SnapshotID    string   `json:"snapshot_id,omitempty"`
	Detail        string   `json:"detail,omitempty"`
	CapturedPaths []string `json:"captured_paths,omitempty"`
}

// ConfigDiff is a read-only comparison between repository and live dotfiles.
type ConfigDiff struct {
	ConfigID string `json:"config_id"`
	Target   string `json:"target"`
	State    string `json:"state"` // clean|missing|drift|conflict
	Summary  string `json:"summary,omitempty"`
	Unified  string `json:"unified,omitempty"`
}

// SyncService manages dotfile deployment, true 3-way merge, capture and diff.
type SyncService struct {
	platform  *platform.Info
	catalog   *catalog.Registry
	snapshots *snapshot.Manager
	store     *state.Store
	repoRoot  string
}

// NewSyncService creates a SyncService; store may be nil (no persisted merge ancestors).
func NewSyncService(
	plat *platform.Info,
	cat *catalog.Registry,
	snap *snapshot.Manager,
	store *state.Store,
	repoRoot string,
) *SyncService {
	return &SyncService{platform: plat, catalog: cat, snapshots: snap, store: store, repoRoot: repoRoot}
}

// SetRepoRoot updates the repository root used for resolving configuration files.
func (s *SyncService) SetRepoRoot(repoRoot string) {
	s.repoRoot = repoRoot
}

// RepoRoot returns the currently configured repository root.
func (s *SyncService) RepoRoot() string {
	return s.repoRoot
}

func (s *SyncService) selectConfigs(group string, overrideShell, overrideTerminal string) []domain.ConfigurationResource {
	all := s.catalog.ListConfigurationsByGroup(group)
	out := make([]domain.ConfigurationResource, 0, len(all))
	sh := s.platform.CurrentShell()
	if overrideShell != "" {
		sh = overrideShell
	}
	if sh == "" {
		sh = "zsh"
	}
	term := s.platform.CurrentTerminal()
	if overrideTerminal != "" {
		term = overrideTerminal
	}
	if term == "" {
		term = "ghostty"
	}
	for _, c := range all {
		if !c.SupportsOS(s.platform.OS()) {
			continue
		}
		if !c.SupportsDesktop(s.platform.DesktopEnv()) {
			continue
		}
		if !c.SupportsShell(sh) {
			continue
		}
		if !c.SupportsTerminal(term) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func (s *SyncService) paths(cfg domain.ConfigurationResource) (source, target string, err error) {
	source, err = security.SafePath(s.repoRoot, cfg.Source)
	if err != nil {
		return "", "", err
	}
	target = s.platform.ExpandPath(cfg.TargetForOS(s.platform.OS()))
	return source, target, nil
}

// Diff compares each selected configuration with its live target.
func (s *SyncService) Diff(_ context.Context, group string) ([]ConfigDiff, error) {
	cfgs := s.selectConfigs(group, "", "")
	diffs := make([]ConfigDiff, 0, len(cfgs))
	for _, cfg := range cfgs {
		source, target, err := s.paths(cfg)
		if err != nil {
			return nil, err
		}
		d := ConfigDiff{ConfigID: cfg.ID, Target: target, State: "clean"}

		srcInfo, err := os.Stat(source)
		if err != nil {
			d.State = "missing"
			d.Unified = "repository source missing: " + cfg.Source
			diffs = append(diffs, d)
			continue
		}
		if _, err := os.Lstat(target); err != nil {
			d.State = "missing"
			diffs = append(diffs, d)
			continue
		}

		if srcInfo.IsDir() {
			fds, err := configsync.DiffTree(target, source, cfg.Excludes)
			if err != nil {
				return nil, err
			}
			for _, fd := range fds {
				d.State = "drift"
				d.Unified += fmt.Sprintf("%s: %s\n", fd.Status, fd.RelPath)
			}
			if len(fds) == 0 {
				d.Summary = "up to date"
			} else {
				d.Summary = fmt.Sprintf("%d file(s) differ", len(fds))
			}
			diffs = append(diffs, d)
			continue
		}

		incoming, err := os.ReadFile(source)
		if err != nil {
			d.State = "missing"
			d.Summary = fmt.Sprintf("source unreadable: %v", err)
			diffs = append(diffs, d)
			continue
		}
		local, err := os.ReadFile(target)
		if err != nil {
			d.State = "missing"
			d.Summary = fmt.Sprintf("target unreadable: %v", err)
			diffs = append(diffs, d)
			continue
		}
		if cfg.Strategy == domain.StrategyMarkerBlock || cfg.Strategy == domain.StrategySmartAppend {
			block, ok := configsync.ExtractMarkerBlock(string(local), markerStart(cfg), markerEnd(cfg))
			if ok && bytes.Equal(bytes.TrimSpace([]byte(block)), bytes.TrimSpace(incoming)) {
				d.Summary = "up to date"
				diffs = append(diffs, d)
				continue
			}
			d.State = "drift"
			d.Unified = configsync.UnifiedDiff("live", "repo", block, string(incoming), 3)
			d.Summary = summarizeDiffLines(d.Unified)
			diffs = append(diffs, d)
			continue
		}

		if bytes.Equal(local, incoming) {
			d.Summary = "up to date"
			diffs = append(diffs, d)
			continue
		}
		if semEq, err := configsync.IsSemanticEqual(target, string(local), string(incoming)); err == nil && semEq {
			d.Summary = "semantically up to date"
			diffs = append(diffs, d)
			continue
		}
		d.State = "drift"
		d.Unified = configsync.UnifiedDiff(target, cfg.Source, string(local), string(incoming), 3)
		if base, ok := s.getBase(cfg.ID); ok {
			if m := configsync.ThreeWayMerge(target, base, string(local), string(incoming)); m.HasConflict {
				d.State = "conflict"
			}
		}
		d.Summary = summarizeDiffLines(d.Unified)
		diffs = append(diffs, d)
	}
	return diffs, nil
}

func summarizeDiffLines(unified string) string {
	added, removed := 0, 0
	for _, line := range strings.Split(unified, "\n") {
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}
		if strings.HasPrefix(line, "+") {
			added++
		} else if strings.HasPrefix(line, "-") {
			removed++
		}
	}
	if added == 0 && removed == 0 {
		return "drift detected"
	}
	return fmt.Sprintf("+%d / -%d lines", added, removed)
}

// Sync deploys (or with Push captures) every selected configuration.
func (s *SyncService) Sync(ctx context.Context, opts SyncOptions) ([]SyncResult, error) {
	cfgs := s.selectConfigs(opts.Group, opts.Shell, opts.Terminal)
	results := make([]SyncResult, 0, len(cfgs))

	if opts.Push {
		for _, cfg := range cfgs {
			capturedPaths, err := configsync.CaptureToRepoDetailed(s.platform.HomeDir(), s.repoRoot, cfg, s.platform.OS(), opts.DryRun)
			if err != nil {
				return results, fmt.Errorf("capturing %s: %w", cfg.ID, err)
			}
			act := "captured"
			if opts.DryRun {
				if len(capturedPaths) > 0 {
					act = "drift"
				} else {
					act = "unchanged"
				}
			} else if len(capturedPaths) == 0 {
				act = "skipped"
			}
			results = append(results, SyncResult{
				ConfigID:      cfg.ID,
				Action:        act,
				Files:         len(capturedPaths),
				CapturedPaths: capturedPaths,
			})
		}
		return results, nil
	}

	snapID := ""
	if !opts.DryRun && s.snapshots != nil {
		var existing []string
		for _, cfg := range cfgs {
			if cfg.Backup {
				if _, target, err := s.paths(cfg); err == nil {
					if _, err := os.Lstat(target); err == nil {
						existing = append(existing, target)
					}
				}
			}
		}
		if len(existing) > 0 {
			snap, err := s.snapshots.CreateSnapshot("pre-sync", existing)
			if err != nil {
				return nil, fmt.Errorf("pre-sync snapshot: %w", err)
			}
			snapID = snap.ID
		}
	}

	for _, cfg := range cfgs {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		res, err := s.syncConfig(cfg, opts)
		if err != nil {
			return results, fmt.Errorf("syncing %s: %w", cfg.ID, err)
		}
		res.SnapshotID = snapID
		results = append(results, res)
	}
	return results, nil
}

// SyncOne synchronizes a single configuration by ID (used by InstallService plan steps).
func (s *SyncService) SyncOne(ctx context.Context, id string, opts SyncOptions) (SyncResult, error) {
	cfg, ok := s.catalog.GetConfiguration(id)
	if !ok {
		return SyncResult{}, fmt.Errorf("unknown configuration %q", id)
	}
	snapID := ""
	if !opts.DryRun && s.snapshots != nil && cfg.Backup {
		if _, target, err := s.paths(cfg); err == nil {
			if _, err := os.Lstat(target); err == nil {
				snap, err := s.snapshots.CreateSnapshot("pre-sync:"+id, []string{target})
				if err != nil {
					return SyncResult{}, err
				}
				snapID = snap.ID
			}
		}
	}
	if err := ctx.Err(); err != nil {
		return SyncResult{}, err
	}
	res, err := s.syncConfig(cfg, opts)
	res.SnapshotID = snapID
	return res, err
}

func (s *SyncService) syncConfig(cfg domain.ConfigurationResource, opts SyncOptions) (SyncResult, error) {
	source, target, err := s.paths(cfg)
	if err != nil {
		return SyncResult{}, err
	}
	res := SyncResult{ConfigID: cfg.ID, Target: target}

	info, err := os.Stat(source)
	if err != nil {
		res.Action = "skipped"
		res.Detail = "repository source not found"
		return res, nil
	}

	var r SyncResult
	var syncErr error

	switch {
	case info.IsDir():
		r, syncErr = s.syncDirectory(cfg, source, target, opts, res)
	case cfg.Strategy == domain.StrategyMarkerBlock || cfg.Strategy == domain.StrategySmartAppend:
		r, syncErr = s.syncMarker(cfg, source, target, opts, res)
	case cfg.Strategy == domain.StrategySymlink:
		if !opts.DryRun {
			if err := configsync.NewDeployer().Deploy(source, target, configsync.ModeSymlink); err != nil {
				return res, err
			}
		}
		r = res
		r.Action = "updated"
	default:
		r, syncErr = s.syncFile(cfg, source, target, opts, res)
	}

	if syncErr == nil && !opts.DryRun && (r.Action == "created" || r.Action == "updated" || r.Action == "merged") {
		if err := s.runPostSync(cfg); err != nil {
			r.Action = "partial"
			if r.Detail != "" {
				r.Detail += "; "
			}
			r.Detail += fmt.Sprintf("post-sync error: %v", err)
			syncErr = fmt.Errorf("post-sync failed for config %q: %w", cfg.ID, err)
		}
	}
	return r, syncErr
}

func (s *SyncService) runPostSync(cfg domain.ConfigurationResource) error {
	var errs []error
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	runCmd := func(cmdArgs []string, prefix string) {
		if len(cmdArgs) == 0 {
			return
		}
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			errOutput := strings.TrimSpace(stderr.String())
			if errOutput != "" {
				errs = append(errs, fmt.Errorf("%scmd %v failed: %w (stderr: %s)", prefix, cmdArgs, err, errOutput))
			} else {
				errs = append(errs, fmt.Errorf("%scmd %v failed: %w", prefix, cmdArgs, err))
			}
		}
	}

	for _, cmdArgs := range cfg.PostSyncCmds {
		runCmd(cmdArgs, "post-sync ")
	}
	if s.platform != nil && len(cfg.PostSyncByDesktop) > 0 {
		de := s.platform.DesktopEnv()
		if deCmds, ok := cfg.PostSyncByDesktop[de]; ok {
			for _, cmdArgs := range deCmds {
				runCmd(cmdArgs, fmt.Sprintf("post-sync desktop (%s) ", de))
			}
		}
	}
	return errors.Join(errs...)
}

func (s *SyncService) syncDirectory(
	cfg domain.ConfigurationResource, source, target string, opts SyncOptions, res SyncResult,
) (SyncResult, error) {
	if opts.DryRun {
		res.Action = "updated"
		return res, nil
	}
	n, err := configsync.NewDeployer().SyncDirectory(source, target, cfg.Excludes)
	if err != nil {
		return res, err
	}
	res.Files = n
	res.Action = "updated"
	if n == 0 {
		res.Action = "unchanged"
	}
	return res, nil
}

func (s *SyncService) syncMarker(
	cfg domain.ConfigurationResource, source, target string, opts SyncOptions, res SyncResult,
) (SyncResult, error) {
	block, err := os.ReadFile(source)
	if err != nil {
		return res, err
	}
	current, _ := os.ReadFile(target)
	updated := configsync.SmartAppend(string(current), string(block), markerStart(cfg), markerEnd(cfg))
	if updated == string(current) {
		res.Action = "unchanged"
		return res, nil
	}
	res.Action = "updated"
	if opts.DryRun {
		return res, nil
	}
	return res, writeFileAtomic(target, []byte(updated), 0o644)
}

func (s *SyncService) syncFile(
	cfg domain.ConfigurationResource, source, target string, opts SyncOptions, res SyncResult,
) (SyncResult, error) {
	incoming, err := os.ReadFile(source)
	if err != nil {
		return res, err
	}
	local, readErr := os.ReadFile(target)
	if errors.Is(readErr, os.ErrNotExist) {
		res.Action = "created"
		if opts.DryRun {
			return res, nil
		}
		if err := writeFileAtomic(target, incoming, fileMode(source)); err != nil {
			return res, err
		}
		s.recordBase(cfg.ID, incoming)
		return res, nil
	}
	if readErr != nil {
		return res, readErr
	}
	if bytes.Equal(local, incoming) {
		res.Action = "unchanged"
		s.recordBase(cfg.ID, incoming)
		return res, nil
	}

	final := incoming
	res.Action = "updated"

	if cfg.Strategy == domain.StrategyThreeWay && !opts.Force {
		if base, ok := s.getBase(cfg.ID); ok {
			m := configsync.ThreeWayMerge(target, base, string(local), string(incoming))
			if m.HasConflict {
				res.Action = "conflict"
				res.Detail = "both sides modified; re-run with --force to take repository version"
				return res, nil
			}
			final = []byte(m.MergedContent)
			if m.ConflictType != domain.ConflictIncomingModified {
				res.Action = "merged"
			}
		}
		// No recorded ancestor: first adoption — repository wins after the pre-sync snapshot.
	}

	if opts.DryRun {
		return res, nil
	}
	if err := writeFileAtomic(target, final, fileMode(source)); err != nil {
		return res, err
	}
	s.recordBase(cfg.ID, incoming)
	return res, nil
}

func (s *SyncService) getBase(id string) (string, bool) {
	if s.store == nil {
		return "", false
	}
	b, ok := s.store.GetConfigBase(id)
	return string(b), ok
}

func (s *SyncService) recordBase(id string, content []byte) {
	if s.store != nil {
		_ = s.store.RecordConfigBase(id, content)
	}
}

func markerStart(cfg domain.ConfigurationResource) string {
	if cfg.MarkerStart != "" {
		return cfg.MarkerStart
	}
	return configsync.DefaultMarkerStart
}

func markerEnd(cfg domain.ConfigurationResource) string {
	if cfg.MarkerEnd != "" {
		return cfg.MarkerEnd
	}
	return configsync.DefaultMarkerEnd
}

func fileMode(path string) os.FileMode {
	if info, err := os.Stat(path); err == nil && info.Mode()&0o111 != 0 {
		return 0o755
	}
	return 0o644
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".anx-sync-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}
