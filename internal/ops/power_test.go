package ops

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupMockPowerFS(t *testing.T, cmdline string, memSleep string, activeVT string, meminfo string, gpuDriver string) (string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	sysRoot := filepath.Join(tmpDir, "sys")
	procRoot := filepath.Join(tmpDir, "proc")

	if err := os.MkdirAll(filepath.Join(procRoot), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sysRoot, "power"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sysRoot, "devices", "virtual", "tty", "tty0"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(sysRoot, "bus", "pci", "drivers"), 0o755); err != nil {
		t.Fatal(err)
	}

	if cmdline != "" {
		_ = os.WriteFile(filepath.Join(procRoot, "cmdline"), []byte(cmdline), 0o644)
	}
	if memSleep != "" {
		_ = os.WriteFile(filepath.Join(sysRoot, "power", "mem_sleep"), []byte(memSleep), 0o644)
	}
	if activeVT != "" {
		_ = os.WriteFile(filepath.Join(sysRoot, "devices", "virtual", "tty", "tty0", "active"), []byte(activeVT), 0o644)
	}
	if meminfo != "" {
		_ = os.WriteFile(filepath.Join(procRoot, "meminfo"), []byte(meminfo), 0o644)
	}
	if gpuDriver != "" {
		_ = os.MkdirAll(filepath.Join(sysRoot, "bus", "pci", "drivers", gpuDriver), 0o755)
	}

	return sysRoot, procRoot
}

func TestDetectStatus_NouveauKeplerQuirk(t *testing.T) {
	meminfo := "MemTotal:       24584284 kB\nSwapTotal:       4194300 kB\n"
	sysRoot, procRoot := setupMockPowerFS(t,
		"BOOT_IMAGE=/vmlinuz root=UUID=123 nouveau.pstate=1 ro quiet splash",
		"s2idle [deep]",
		"tty7\n",
		meminfo,
		"nouveau",
	)

	var buf bytes.Buffer
	pm := NewPowerManager(nil, nil,
		WithPowerRoots(sysRoot, procRoot),
		WithPowerOutput(&buf),
	)

	status, err := pm.DetectStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.GPUDriver != "nouveau" {
		t.Errorf("expected gpu driver nouveau, got %q", status.GPUDriver)
	}
	if !status.HasKeplerPStateBug {
		t.Errorf("expected HasKeplerPStateBug = true")
	}
	if status.ActiveVT != "7" {
		t.Errorf("expected active vt '7', got %q", status.ActiveVT)
	}
	if status.RAMTotalMB != 24008 {
		t.Errorf("expected RAM 24008 MB, got %d", status.RAMTotalMB)
	}
	if status.SwapTotalMB != 4095 {
		t.Errorf("expected Swap 4095 MB, got %d", status.SwapTotalMB)
	}
	if status.CanHibernate {
		t.Errorf("expected CanHibernate = false since swap < ram")
	}
	if !strings.Contains(status.RecommendedStrategy, "lock") {
		t.Errorf("expected recommendation to suggest lock, got %q", status.RecommendedStrategy)
	}

	pm.PrintStatus(status)
	out := buf.String()
	if !strings.Contains(out, "nouveau.pstate=1 detected") {
		t.Errorf("expected report to mention nouveau.pstate=1, got:\n%s", out)
	}
}

func TestDetectStatus_S2Idle(t *testing.T) {
	meminfo := "MemTotal:        8192000 kB\nSwapTotal:      16384000 kB\n"
	sysRoot, procRoot := setupMockPowerFS(t,
		"BOOT_IMAGE=/vmlinuz ro quiet",
		"[s2idle] deep",
		"tty1\n",
		meminfo,
		"i915",
	)

	var buf bytes.Buffer
	pm := NewPowerManager(nil, nil,
		WithPowerRoots(sysRoot, procRoot),
		WithPowerOutput(&buf),
	)

	status, err := pm.DetectStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.GPUDriver != "i915" {
		t.Errorf("expected gpu driver i915, got %q", status.GPUDriver)
	}
	if status.HasKeplerPStateBug {
		t.Errorf("expected HasKeplerPStateBug = false")
	}
	if !status.CanHibernate {
		t.Errorf("expected CanHibernate = true (swap 16GB >= ram 8GB)")
	}
	if !strings.Contains(status.RecommendedStrategy, "s2idle") {
		t.Errorf("expected recommendation to suggest s2idle, got %q", status.RecommendedStrategy)
	}
}

func TestExecute_DryRun(t *testing.T) {
	meminfo := "MemTotal:       8192000 kB\nSwapTotal:      8192000 kB\n"
	sysRoot, procRoot := setupMockPowerFS(t,
		"BOOT_IMAGE=/vmlinuz ro quiet",
		"s2idle deep",
		"tty7\n",
		meminfo,
		"nvidia",
	)

	var buf bytes.Buffer
	pm := NewPowerManager(nil, nil,
		WithPowerRoots(sysRoot, procRoot),
		WithPowerOutput(&buf),
		WithPowerDryRun(true),
	)

	ctx := context.Background()

	// Test lock
	buf.Reset()
	if err := pm.Execute(ctx, "lock"); err != nil {
		t.Fatalf("unexpected error in dry-run lock: %v", err)
	}
	if !strings.Contains(buf.String(), "[dry-run]") {
		t.Errorf("expected output to contain [dry-run], got: %s", buf.String())
	}

	// Test s2idle
	buf.Reset()
	if err := pm.Execute(ctx, "s2idle"); err != nil {
		t.Fatalf("unexpected error in dry-run s2idle: %v", err)
	}
	if !strings.Contains(buf.String(), "[dry-run]") {
		t.Errorf("expected output to contain [dry-run], got: %s", buf.String())
	}

	// Test deep VT
	buf.Reset()
	if err := pm.Execute(ctx, "deep"); err != nil {
		t.Fatalf("unexpected error in dry-run deep: %v", err)
	}
	if !strings.Contains(buf.String(), "[dry-run]") {
		t.Errorf("expected output to contain [dry-run], got: %s", buf.String())
	}

	// Test hibernate
	buf.Reset()
	if err := pm.Execute(ctx, "hibernate"); err != nil {
		t.Fatalf("unexpected error in dry-run hibernate: %v", err)
	}
	if !strings.Contains(buf.String(), "[dry-run]") {
		t.Errorf("expected output to contain [dry-run], got: %s", buf.String())
	}

	// Test status
	buf.Reset()
	if err := pm.Execute(ctx, "status"); err != nil {
		t.Fatalf("unexpected error in status: %v", err)
	}
	if !strings.Contains(buf.String(), "ANX Safe Power & Suspend Diagnostics") {
		t.Errorf("expected status header, got: %s", buf.String())
	}

	// Test auto
	buf.Reset()
	if err := pm.Execute(ctx, ""); err != nil {
		t.Fatalf("unexpected error in auto: %v", err)
	}
	if !strings.Contains(buf.String(), "Safe Screen Standby") {
		t.Errorf("expected auto to select safe standby, got: %s", buf.String())
	}

	// Test invalid
	if err := pm.Execute(ctx, "bogus-mode"); err == nil {
		t.Errorf("expected error for bogus mode")
	}
}

func TestHibernate_InsufficientSwap(t *testing.T) {
	meminfo := "MemTotal:       16384000 kB\nSwapTotal:      2048000 kB\n"
	sysRoot, procRoot := setupMockPowerFS(t, "", "deep", "tty1", meminfo, "")

	var buf bytes.Buffer
	pm := NewPowerManager(nil, nil,
		WithPowerRoots(sysRoot, procRoot),
		WithPowerOutput(&buf),
	)

	err := pm.Hibernate(context.Background())
	if err == nil {
		t.Fatalf("expected error when swap < ram")
	}
	if !strings.Contains(err.Error(), "insufficient swap space") {
		t.Errorf("expected insufficient swap space error, got: %v", err)
	}
}
