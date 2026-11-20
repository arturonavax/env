// Package osconfig snapshots and restores the macOS state altered by scripts/osconfig/macos.sh.
//
// Snapshots are domain-level: every preference domain the script writes is exported with
// `defaults export` and restored with `defaults import`, which also removes keys that did not
// exist before the script ran. System-level pieces (/etc/pam.d/sudo_local, pmset lowpowermode,
// the loginwindow domain) are captured too and restored through sudo.
package osconfig

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// DefaultKeep is the total number of snapshots retained (the first one always included).
const DefaultKeep = 5

// ExecFunc runs a command and returns its stdout (injectable for tests).
type ExecFunc func(ctx context.Context, name string, args ...string) ([]byte, error)

// DomainSpec describes one preference domain altered by the osconfig script.
type DomainSpec struct {
	Domain      string `json:"domain"`
	CurrentHost bool   `json:"current_host,omitempty"`
	System      bool   `json:"system,omitempty"` // needs sudo to restore
	Optional    bool   `json:"optional,omitempty"`
}

// ManagedDomains lists every domain written by scripts/osconfig/macos.sh.
// A unit test keeps this list in sync with the script.
func ManagedDomains(home string) []DomainSpec {
	user := []string{
		"NSGlobalDomain",
		"com.apple.dock",
		"com.apple.finder",
		"com.apple.AppleMultitouchTrackpad",
		"com.apple.driver.AppleBluetoothMultitouch.trackpad",
		"com.apple.universalaccess",
		"com.apple.menuextra.clock",
		"com.apple.ActivityMonitor",
		"com.google.Chrome",
		"com.apple.TextEdit",
		"com.apple.Terminal",
		"com.apple.WindowManager",
		"com.apple.SoftwareUpdate",
		"com.mitchellh.ghostty",
		"com.apple.screencapture",
		"com.apple.DiskUtility",
		"com.apple.spaces",
		"com.apple.print.PrintingPrefs",
		"com.apple.helpviewer",
		"com.apple.dt.Xcode",
		"com.apple.desktopservices",
		"com.apple.TimeMachine",
		"com.apple.Siri",
		"com.apple.symbolichotkeys",
		"com.apple.Safari.SandboxBroker",
		"com.apple.QuickTimePlayerX",
		"com.apple.ImageCapture",
		"com.apple.Music",
	}
	var specs []DomainSpec
	for _, d := range user {
		specs = append(specs, DomainSpec{Domain: d})
	}
	specs = append(specs,
		DomainSpec{Domain: "NSGlobalDomain", CurrentHost: true},
		DomainSpec{Domain: "com.apple.controlcenter", CurrentHost: true},
		DomainSpec{Domain: filepath.Join(home, "Library/Containers/com.apple.Safari/Data/Library/Preferences/com.apple.Safari"), Optional: true},
		DomainSpec{Domain: "/Library/Preferences/com.apple.loginwindow", System: true, Optional: true},
	)
	return specs
}

// DomainEntry is a captured domain inside a snapshot.
type DomainEntry struct {
	DomainSpec
	File string `json:"file"` // relative to the snapshot directory
}

// Manifest describes one snapshot.
type Manifest struct {
	ID           string        `json:"id"`
	Label        string        `json:"label"`
	CreatedAt    time.Time     `json:"created_at"`
	MacOSVersion string        `json:"macos_version,omitempty"`
	Fingerprint  string        `json:"fingerprint"`
	Domains      []DomainEntry `json:"domains"`
	Skipped      []string      `json:"skipped,omitempty"`
	SudoLocal    string        `json:"sudo_local,omitempty"`     // relative file; empty when it did not exist
	SudoLocalSet bool          `json:"sudo_local_set"`           // true when the file was inspected
	LowPowerMode string        `json:"low_power_mode,omitempty"` // "0"/"1"; empty when unknown
	First        bool          `json:"-"`
}

// Manager creates, lists, prunes, and restores snapshots.
type Manager struct {
	Root          string // directory holding one sub-directory per snapshot
	Home          string
	SudoLocalPath string
	Keep          int
	Exec          ExecFunc
	Now           func() time.Time
}

// NewManager builds a Manager with production defaults.
func NewManager(root, home string) *Manager {
	return &Manager{
		Root:          root,
		Home:          home,
		SudoLocalPath: "/etc/pam.d/sudo_local",
		Keep:          DefaultKeep,
		Exec:          defaultExec,
		Now:           time.Now,
	}
}

func (m *Manager) keep() int {
	if m.Keep < 2 {
		return DefaultKeep
	}
	return m.Keep
}

func (m *Manager) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func domainFileName(i int, s DomainSpec) string {
	name := unsafeName.ReplaceAllString(strings.Trim(s.Domain, "/"), "_")
	if s.CurrentHost {
		name += ".currenthost"
	}
	return fmt.Sprintf("%02d-%s.plist", i, name)
}

func exportArgs(s DomainSpec) []string {
	args := []string{"defaults"}
	if s.CurrentHost {
		args = append(args, "-currentHost")
	}
	return append(args, "export", s.Domain, "-")
}

// Create snapshots the current state. When the state is identical to the newest snapshot, that
// snapshot is returned instead of storing a duplicate. prune enforces the retention policy.
func (m *Manager) Create(ctx context.Context, label string, prune bool) (*Manifest, bool, error) {
	if m.Exec == nil {
		return nil, false, errors.New("osconfig: no command runner configured")
	}
	if err := os.MkdirAll(m.Root, 0o700); err != nil {
		return nil, false, fmt.Errorf("creating backup root: %w", err)
	}
	base := m.now().UTC().Format("20060102-150405")
	id := base
	dir := filepath.Join(m.Root, id)
	for n := 2; ; n++ {
		if err := os.Mkdir(dir, 0o700); err == nil {
			break
		} else if !errors.Is(err, os.ErrExist) {
			return nil, false, fmt.Errorf("creating snapshot dir: %w", err)
		}
		id = fmt.Sprintf("%s-%d", base, n)
		dir = filepath.Join(m.Root, id)
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(dir)
		}
	}()

	man := &Manifest{ID: id, Label: label, CreatedAt: m.now().UTC()}
	if out, err := m.Exec(ctx, "sw_vers", "-productVersion"); err == nil {
		man.MacOSVersion = strings.TrimSpace(string(out))
	}
	h := sha256.New()

	for i, spec := range ManagedDomains(m.Home) {
		args := exportArgs(spec)
		out, err := m.Exec(ctx, args[0], args[1:]...)
		if err != nil || len(out) == 0 {
			man.Skipped = append(man.Skipped, spec.Domain)
			continue
		}
		file := domainFileName(i, spec)
		if err := os.WriteFile(filepath.Join(dir, file), out, 0o600); err != nil {
			return nil, false, fmt.Errorf("writing %s: %w", file, err)
		}
		man.Domains = append(man.Domains, DomainEntry{DomainSpec: spec, File: file})
		fmt.Fprintf(h, "%s|%t|", spec.Domain, spec.CurrentHost)
		h.Write(out)
	}

	man.SudoLocalSet = true
	if data, err := os.ReadFile(m.SudoLocalPath); err == nil {
		if err := os.WriteFile(filepath.Join(dir, "sudo_local"), data, 0o600); err != nil {
			return nil, false, err
		}
		man.SudoLocal = "sudo_local"
		h.Write([]byte("sudo_local|"))
		h.Write(data)
	} else if !errors.Is(err, os.ErrNotExist) {
		man.SudoLocalSet = false
		man.Skipped = append(man.Skipped, m.SudoLocalPath)
	}
	if out, err := m.Exec(ctx, "pmset", "-g"); err == nil {
		if mt := regexp.MustCompile(`(?m)^\s*lowpowermode\s+(\d)`).FindSubmatch(out); mt != nil {
			man.LowPowerMode = string(mt[1])
			h.Write([]byte("lpm|" + man.LowPowerMode))
		}
	}
	man.Fingerprint = hex.EncodeToString(h.Sum(nil))

	if existing, err := m.List(); err == nil && len(existing) > 0 {
		if last := existing[len(existing)-1]; last.Fingerprint == man.Fingerprint {
			return last, false, nil
		}
	}

	raw, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return nil, false, err
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o600); err != nil {
		return nil, false, err
	}
	cleanup = false
	if prune {
		if err := m.Prune(); err != nil {
			return man, true, fmt.Errorf("pruning old snapshots: %w", err)
		}
	}
	return man, true, nil
}

// List returns snapshots oldest first; the first entry has First set.
func (m *Manager) List() ([]*Manifest, error) {
	entries, err := os.ReadDir(m.Root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []*Manifest
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(m.Root, e.Name(), "manifest.json"))
		if err != nil {
			continue
		}
		var man Manifest
		if json.Unmarshal(raw, &man) != nil || man.ID == "" {
			continue
		}
		out = append(out, &man)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > 0 {
		out[0].First = true
	}
	return out, nil
}

// Prune keeps the first (original) snapshot plus the newest ones, DefaultKeep in total.
func (m *Manager) Prune() error {
	all, err := m.List()
	if err != nil || len(all) <= m.keep() {
		return err
	}
	// all[0] is the baseline; keep it and the (keep-1) newest.
	removable := all[1 : len(all)-(m.keep()-1)]
	var errs []error
	for _, man := range removable {
		if err := os.RemoveAll(filepath.Join(m.Root, man.ID)); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Resolve maps "first", "latest" or an exact ID to a snapshot.
func (m *Manager) Resolve(ref string) (*Manifest, error) {
	all, err := m.List()
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, errors.New("no osconfig backups found")
	}
	switch strings.ToLower(strings.TrimSpace(ref)) {
	case "first", "original":
		return all[0], nil
	case "", "latest":
		return all[len(all)-1], nil
	}
	for _, man := range all {
		if man.ID == ref {
			return man, nil
		}
	}
	return nil, fmt.Errorf("backup %q not found (use 'anx osconfig list')", ref)
}

// RestoreReport summarizes a restore run.
type RestoreReport struct {
	Restored   []string
	Warnings   []string
	SafetyCopy string
	Snapshot   *Manifest
	NeedsSudo  bool
}

// Restore reverts the system to a snapshot. Unless skipSafety is set, the current state is
// snapshotted first (label "pre-restore") so the restore itself can be undone.
func (m *Manager) Restore(ctx context.Context, ref string, skipSafety, restartServices bool) (*RestoreReport, error) {
	man, err := m.Resolve(ref)
	if err != nil {
		return nil, err
	}
	rep := &RestoreReport{Snapshot: man}
	dir := filepath.Join(m.Root, man.ID)

	if !skipSafety {
		if safety, created, err := m.Create(ctx, "pre-restore", false); err != nil {
			return nil, fmt.Errorf("creating safety backup before restore: %w", err)
		} else if created {
			rep.SafetyCopy = safety.ID
		}
	}

	for _, d := range man.Domains {
		file := filepath.Join(dir, d.File)
		args := []string{"defaults"}
		if d.CurrentHost {
			args = append(args, "-currentHost")
		}
		args = append(args, "import", d.Domain, file)
		name := args[0]
		if d.System {
			args = append([]string{"defaults"}, args[1:]...)
			name = "sudo"
			args = append([]string{"-n"}, args...)
		} else {
			args = args[1:]
		}
		label := d.Domain
		if d.CurrentHost {
			label += " (currentHost)"
		}
		if d.System {
			rep.NeedsSudo = true
		}
		if _, err := m.Exec(ctx, name, args...); err != nil {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("could not restore %s: %v", label, err))
			continue
		}
		rep.Restored = append(rep.Restored, label)
	}

	if man.SudoLocalSet {
		rep.NeedsSudo = true
		if man.SudoLocal != "" {
			_, err = m.Exec(ctx, "sudo", "-n", "install", "-m", "0644", filepath.Join(dir, man.SudoLocal), m.SudoLocalPath)
		} else {
			_, err = m.Exec(ctx, "sudo", "-n", "rm", "-f", m.SudoLocalPath)
		}
		if err != nil {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("could not restore %s: %v", m.SudoLocalPath, err))
		} else {
			rep.Restored = append(rep.Restored, m.SudoLocalPath)
		}
	}
	if man.LowPowerMode != "" {
		rep.NeedsSudo = true
		if _, err := m.Exec(ctx, "sudo", "-n", "pmset", "-a", "lowpowermode", man.LowPowerMode); err != nil {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("could not restore pmset lowpowermode: %v", err))
		} else {
			rep.Restored = append(rep.Restored, "pmset lowpowermode")
		}
	}

	if restartServices {
		_, _ = m.Exec(ctx, "killall", "cfprefsd")
		for _, app := range []string{"Dock", "Finder", "SystemUIServer"} {
			_, _ = m.Exec(ctx, "killall", app)
		}
	}
	if err := m.Prune(); err != nil {
		rep.Warnings = append(rep.Warnings, err.Error())
	}
	return rep, nil
}
