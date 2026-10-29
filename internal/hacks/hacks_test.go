package hacks_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/hacks"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
	"github.com/arturonavax/env/internal/snapshot"
	"github.com/arturonavax/env/internal/state"
)

func TestLoadRegistryEmbeddedAndOverlays(t *testing.T) {
	userDropIn := t.TempDir()
	customYAML := `
id: custom-laptop-led
name: Custom Laptop Mute LED Fix
category: hardware-input
description: Toggles custom ALSA mute LED GPIO.
safe_by_opt_in: true
match:
  os: [linux]
  dmi_vendor: Framework
steps:
  - id: fix-mute-led
    name: Fix Mute LED
    command: ["true"]
`
	if err := os.WriteFile(filepath.Join(userDropIn, "framework.yaml"), []byte(customYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	reg, err := hacks.LoadRegistry(
		hacks.WithRepoHacksDir("../../hacks"),
		hacks.WithUserHacksDir(userDropIn),
	)
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	m6800, ok := reg.Get("dell-precision-m6800")
	if !ok {
		t.Fatalf("expected embedded/repo profile 'dell-precision-m6800'")
	}
	if !m6800.SafeByOptIn || m6800.Match.GPUDriver != "nouveau" {
		t.Errorf("unexpected dell-precision-m6800 metadata: %+v", m6800)
	}

	scroll, ok := reg.Get("x11-scroll-lock-led")
	if !ok {
		t.Fatalf("expected embedded/repo profile 'x11-scroll-lock-led'")
	}
	if len(scroll.Match.DisplayServer) != 1 || scroll.Match.DisplayServer[0] != "x11" {
		t.Errorf("unexpected x11-scroll-lock-led matcher: %+v", scroll.Match)
	}

	if _, ok := reg.Get("custom-laptop-led"); !ok {
		t.Errorf("expected user drop-in profile 'custom-laptop-led'")
	}

	// Verify bundled canonical scripts pass security guardrails cleanly.
	for _, scriptRel := range []string{
		"../../hacks/dell-precision-m6800/apply_hacks.sh",
		"../../hacks/x11-scroll-lock-led/activate_scroll_lock.sh",
	} {
		raw, err := os.ReadFile(scriptRel)
		if err != nil {
			t.Fatalf("reading %s: %v", scriptRel, err)
		}
		if violations := security.ValidateScriptGuardrails(string(raw)); len(violations) > 0 {
			t.Errorf("bundled script %s triggered guardrails: %+v", scriptRel, violations)
		}
	}
}

func TestProbeHardwareAndMatchHacks(t *testing.T) {
	sysRoot := t.TempDir()
	procRoot := t.TempDir()

	mustWrite := func(base, rel, content string) {
		t.Helper()
		p := filepath.Join(base, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite(sysRoot, "class/dmi/id/sys_vendor", "Dell Inc.\n")
	mustWrite(sysRoot, "class/dmi/id/product_name", "Precision M6800\n")
	mustWrite(sysRoot, "bus/pci/devices/0000:01:00.0/class", "0x030000\n")
	mustWrite(sysRoot, "bus/pci/devices/0000:01:00.0/uevent", "DRIVER=nouveau\nPCI_CLASS=30000\nPCI_ID=10DE:11FC\n")

	mustWrite(procRoot, "cpuinfo", "processor\t: 0\nmodel name\t: Intel(R) Core(TM) i7-4910MQ CPU @ 2.90GHz\n")
	mustWrite(procRoot, "meminfo", "MemTotal:       32768000 kB\n")
	mustWrite(procRoot, "sys/kernel/osrelease", "6.12.0-generic\n")

	plat := platform.NewTestInfo(platform.TestInfoParams{
		OS:           "linux",
		Arch:         "amd64",
		Distribution: "ubuntu",
		HomeDir:      t.TempDir(),
	})

	hw := hacks.ProbeHardware(
		context.Background(),
		plat,
		nil,
		hacks.WithProbeRoots(sysRoot, procRoot),
		hacks.WithProbeEnv([]string{
			"XDG_CURRENT_DESKTOP=XFCE",
			"XDG_SESSION_TYPE=x11",
			"DISPLAY=:0",
		}),
	)

	if hw.DMIVendor != "Dell Inc." || hw.DMIProduct != "Precision M6800" {
		t.Fatalf("unexpected DMI probe: vendor=%q product=%q", hw.DMIVendor, hw.DMIProduct)
	}
	if hw.DisplayServer != "x11" || hw.DesktopEnv != "xfce" {
		t.Fatalf("unexpected desktop/display probe: desktop=%q display=%q", hw.DesktopEnv, hw.DisplayServer)
	}
	if len(hw.GPUDrivers) == 0 || hw.GPUDrivers[0] != "nouveau" {
		t.Fatalf("expected nouveau GPU driver, got %v", hw.GPUDrivers)
	}

	reg, err := hacks.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	profiles := reg.List()

	t.Run("Dell M6800 Xfce X11 matches both dell-precision-m6800 and x11-scroll-lock-led", func(t *testing.T) {
		matched := hacks.MatchHacks(hw, profiles)
		ids := map[string]bool{}
		for _, p := range matched {
			ids[p.ID] = true
		}
		if !ids["dell-precision-m6800"] {
			t.Errorf("expected dell-precision-m6800 to match, got %v", ids)
		}
		if !ids["x11-scroll-lock-led"] {
			t.Errorf("expected x11-scroll-lock-led to match on X11, got %v", ids)
		}
	})

	t.Run("Wayland GNOME ThinkPad matches neither Dell M6800 nor X11 Scroll Lock", func(t *testing.T) {
		nonMatchHW := domain.HardwareProfile{
			OS:            "linux",
			Distro:        "fedora",
			DMIVendor:     "LENOVO",
			DMIProduct:    "21HM0000US",
			GPUDrivers:    []string{"i915"},
			GPUModels:     []string{"Intel Iris Xe"},
			DesktopEnv:    "gnome",
			DisplayServer: "wayland",
		}
		matched := hacks.MatchHacks(nonMatchHW, profiles)
		if len(matched) != 0 {
			t.Fatalf("expected 0 matched hacks on Wayland ThinkPad, got: %+v", matched)
		}
	})

	t.Run("Container or headless host without /sys/class/dmi probes gracefully without error", func(t *testing.T) {
		emptySys := t.TempDir()
		emptyProc := t.TempDir()
		headlessHW := hacks.ProbeHardware(
			context.Background(),
			plat,
			nil,
			hacks.WithProbeRoots(emptySys, emptyProc),
			hacks.WithProbeEnv([]string{}),
		)
		if headlessHW.DisplayServer != "headless" {
			t.Errorf("expected headless display server, got %q", headlessHW.DisplayServer)
		}
	})
}

func TestPlanHackAndApplyHackIsolatedExecution(t *testing.T) {
	homeDir := t.TempDir()
	stateDir := t.TempDir()
	hackDir := t.TempDir()

	// Seed an existing user config file to verify pre-hack snapshot backup works.
	cfgPath := filepath.Join(homeDir, ".config/demo-app/settings.conf")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("vsync=true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	scriptPath := filepath.Join(hackDir, "apply_demo.sh")
	scriptBody := "#!/usr/bin/env bash\nset -euo pipefail\necho 'vsync=false' > \"$HOME/.config/demo-app/settings.conf\"\n"
	if err := os.WriteFile(scriptPath, []byte(scriptBody), 0o755); err != nil {
		t.Fatal(err)
	}

	profile := domain.HackProfile{
		ID:          "demo-workstation-hack",
		Name:        "Demo Workstation VSync Fix",
		Category:    "workstation",
		SafeByOptIn: true,
		BackupPaths: []string{"~/.config/demo-app/settings.conf"},
		Steps: []domain.HackStep{
			{
				ID:          "disable-vsync",
				Name:        "Disable VSync",
				Description: "Sets vsync=false in demo-app settings",
				Script:      "apply_demo.sh",
				Touches:     []string{"~/.config/demo-app/settings.conf"},
			},
		},
		Dir: hackDir,
	}

	plat := platform.NewTestInfo(platform.TestInfoParams{
		OS:      "linux",
		Arch:    "amd64",
		HomeDir: homeDir,
	})

	plan, err := hacks.PlanHack(plat, profile)
	if err != nil {
		t.Fatalf("PlanHack failed: %v", err)
	}
	if len(plan.BackupPaths) != 1 || plan.BackupPaths[0] != cfgPath {
		t.Errorf("expected expanded backup path %q, got %v", cfgPath, plan.BackupPaths)
	}
	if len(plan.Steps) != 1 || !strings.Contains(plan.Steps[0].CommandLine, "apply_demo.sh") {
		t.Errorf("unexpected planned steps: %+v", plan.Steps)
	}

	snaps, err := snapshot.NewManager(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	store, err := state.OpenStore(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	runner := security.NewRunner(10 * time.Second)

	res, err := hacks.ApplyHack(context.Background(), profile, hacks.ApplyOptions{
		Platform:  plat,
		Runner:    runner,
		Snapshots: snaps,
		Store:     store,
	})
	if err != nil {
		t.Fatalf("ApplyHack failed: %v", err)
	}
	if res.SnapshotID == "" {
		t.Errorf("expected pre-hack snapshot ID to be recorded")
	}
	if !store.IsHackApplied("demo-workstation-hack") {
		t.Errorf("expected store.IsHackApplied('demo-workstation-hack') to be true")
	}

	updated, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(updated)) != "vsync=false" {
		t.Fatalf("expected hack script to update settings.conf, got %q", string(updated))
	}

	// Verify rollback from pre-hack snapshot restores original vsync=true!
	if err := snaps.Restore(res.SnapshotID); err != nil {
		t.Fatalf("restoring pre-hack snapshot failed: %v", err)
	}
	restored, _ := os.ReadFile(cfgPath)
	if strings.TrimSpace(string(restored)) != "vsync=true" {
		t.Fatalf("expected snapshot rollback to restore 'vsync=true', got %q", string(restored))
	}

	t.Run("blocks hack script containing security guardrail violation", func(t *testing.T) {
		badDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(badDir, "evil.sh"), []byte("#!/usr/bin/env bash\nexit 0\nrm -rf /etc\nmkfs.ext4 /dev/sda1\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		badProfile := domain.HackProfile{
			ID:   "evil-hack",
			Name: "Evil Hack",
			Steps: []domain.HackStep{
				{ID: "wipe", Name: "Wipe", Script: "evil.sh"},
			},
			Dir: badDir,
		}
		_, applyErr := hacks.ApplyHack(context.Background(), badProfile, hacks.ApplyOptions{
			Platform: plat,
			Runner:   runner,
		})
		if !errors.Is(applyErr, hacks.ErrHackGuardrailViolation) {
			t.Fatalf("expected ErrHackGuardrailViolation, got: %v", applyErr)
		}
	})
}
