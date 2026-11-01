package ops

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
)

// PowerMode defines the operational mode for power management.
type PowerMode string

const (
	PowerModeStatus    PowerMode = "status"
	PowerModeLock      PowerMode = "lock"
	PowerModeS2Idle    PowerMode = "s2idle"
	PowerModeDeepVT    PowerMode = "deep"
	PowerModeHibernate PowerMode = "hibernate"
	PowerModeAuto      PowerMode = "auto"
)

// PowerStatus contains hardware and kernel diagnostics relevant to power management.
type PowerStatus struct {
	GPUDriver           string `json:"gpu_driver"`
	HasKeplerPStateBug  bool   `json:"has_kepler_pstate_bug"`
	MemSleepModes       string `json:"mem_sleep_modes"`
	ActiveVT            string `json:"active_vt"`
	RAMTotalMB          uint64 `json:"ram_total_mb"`
	SwapTotalMB         uint64 `json:"swap_total_mb"`
	CanHibernate        bool   `json:"can_hibernate"`
	RecommendedStrategy string `json:"recommended_strategy"`
}

// PowerOption configures a PowerManager instance.
type PowerOption func(*PowerManager)

// WithPowerDryRun enables dry-run preview mode.
func WithPowerDryRun(dryRun bool) PowerOption {
	return func(p *PowerManager) {
		p.dryRun = dryRun
	}
}

// WithPowerOutput redirects log and status output.
func WithPowerOutput(w io.Writer) PowerOption {
	return func(p *PowerManager) {
		if w != nil {
			p.out = w
		}
	}
}

// WithPowerRoots overrides /sys and /proc root paths for deterministic testing.
func WithPowerRoots(sysRoot, procRoot string) PowerOption {
	return func(p *PowerManager) {
		if sysRoot != "" {
			p.sysRoot = sysRoot
		}
		if procRoot != "" {
			p.procRoot = procRoot
		}
	}
}

// PowerManager provides hardware-aware power transitions (lock, s2idle, deep VT, hibernate).
type PowerManager struct {
	platform *platform.Info
	runner   *security.Runner
	dryRun   bool
	out      io.Writer
	sysRoot  string
	procRoot string
}

// NewPowerManager creates a new PowerManager instance.
func NewPowerManager(plat *platform.Info, runner *security.Runner, opts ...PowerOption) *PowerManager {
	pm := &PowerManager{
		platform: plat,
		runner:   runner,
		out:      os.Stdout,
		sysRoot:  "/sys",
		procRoot: "/proc",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(pm)
		}
	}
	return pm
}

func (pm *PowerManager) log(format string, a ...any) {
	fmt.Fprintf(pm.out, format+"\n", a...)
}

func (pm *PowerManager) logStep(msg string) {
	if pm.dryRun {
		pm.log("  [dry-run] %s", msg)
	} else {
		pm.log("  [STEP] %s", msg)
	}
}

func (pm *PowerManager) logSuccess(msg string) {
	pm.log("  [OK] %s", msg)
}

func (pm *PowerManager) logWarn(msg string) {
	pm.log("  [WARN] %s", msg)
}

func (pm *PowerManager) logError(msg string) {
	pm.log("  [ERROR] %s", msg)
}

// DetectStatus probes the host hardware, GPU driver, sleep modes, and swap configuration.
func (pm *PowerManager) DetectStatus(ctx context.Context) (*PowerStatus, error) {
	status := &PowerStatus{
		ActiveVT:      "7",
		MemSleepModes: "unknown",
	}

	// 1. Detect GPU Driver
	status.GPUDriver = pm.detectGPUDriver(ctx)

	// 2. Check for Kepler nouveau.pstate=1 quirk
	cmdlinePath := filepath.Join(pm.procRoot, "cmdline")
	if data, err := os.ReadFile(cmdlinePath); err == nil {
		if strings.Contains(string(data), "nouveau.pstate=1") {
			status.HasKeplerPStateBug = true
		}
	}

	// 3. Inspect ACPI memory sleep states
	memSleepPath := filepath.Join(pm.sysRoot, "power", "mem_sleep")
	if data, err := os.ReadFile(memSleepPath); err == nil {
		status.MemSleepModes = strings.TrimSpace(string(data))
	}

	// 4. Query active virtual terminal
	activeVTPath := filepath.Join(pm.sysRoot, "devices", "virtual", "tty", "tty0", "active")
	if data, err := os.ReadFile(activeVTPath); err == nil {
		vt := strings.TrimSpace(string(data))
		status.ActiveVT = strings.TrimPrefix(vt, "tty")
	}

	// 5. Read RAM and Swap from /proc/meminfo
	meminfoPath := filepath.Join(pm.procRoot, "meminfo")
	if f, err := os.Open(meminfoPath); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				key := strings.TrimSuffix(fields[0], ":")
				valKB, _ := strconv.ParseUint(fields[1], 10, 64)
				switch key {
				case "MemTotal":
					status.RAMTotalMB = valKB / 1024
				case "SwapTotal":
					status.SwapTotalMB = valKB / 1024
				}
			}
		}
	}

	if status.SwapTotalMB >= status.RAMTotalMB && status.RAMTotalMB > 0 {
		status.CanHibernate = true
	}

	// Recommend safest strategy based on hardware profile
	if status.GPUDriver == "nouveau" && status.HasKeplerPStateBug {
		status.RecommendedStrategy = "lock (Screen standby via DPMS; ACPI S3 deep sleep deadlocks Kepler VRAM PLL)"
	} else if strings.Contains(status.MemSleepModes, "s2idle") {
		status.RecommendedStrategy = "s2idle (Suspend-to-idle; CPU low power, leaves GPU PLL intact)"
	} else {
		status.RecommendedStrategy = "lock (Safe screen standby)"
	}

	return status, nil
}

func (pm *PowerManager) detectGPUDriver(ctx context.Context) string {
	drivers := []string{"nouveau", "nvidia", "i915", "iris", "xe", "amdgpu", "radeon"}
	pciDriversDir := filepath.Join(pm.sysRoot, "bus", "pci", "drivers")
	for _, d := range drivers {
		path := filepath.Join(pciDriversDir, d)
		if fi, err := os.Stat(path); err == nil && fi.IsDir() {
			return d
		}
	}

	// Fallback via lspci if runner available
	if pm.runner != nil {
		out, err := pm.runner.Run(ctx, "lspci", "-nnk")
		if err == nil {
			lower := strings.ToLower(out)
			if strings.Contains(lower, "nouveau") {
				return "nouveau"
			}
			if strings.Contains(lower, "nvidia") {
				return "nvidia"
			}
			if strings.Contains(lower, "i915") || strings.Contains(lower, "iris") || strings.Contains(lower, "xe") {
				return "intel"
			}
			if strings.Contains(lower, "amdgpu") || strings.Contains(lower, "radeon") {
				return "amd"
			}
		}
	}

	return "unknown"
}

// PrintStatus writes a human-readable diagnostic report.
func (pm *PowerManager) PrintStatus(status *PowerStatus) {
	pm.log("==============================================================================")
	pm.log("  ANX Safe Power & Suspend Diagnostics")
	pm.log("==============================================================================")
	pm.log("  * GPU Driver in use  : %s", status.GPUDriver)
	if status.HasKeplerPStateBug {
		pm.log("  * Kepler P-State     : nouveau.pstate=1 detected (ACPI S3 deep sleep causes VRAM PLL deadlock)")
	} else {
		pm.log("  * Kepler P-State     : Standard (no manual pstate override detected)")
	}
	pm.log("  * Memory Sleep Modes : %s", status.MemSleepModes)
	pm.log("  * Active X11 Console : tty%s", status.ActiveVT)
	pm.log("  * Physical RAM       : %d MB", status.RAMTotalMB)
	pm.log("  * Swap Allocated     : %d MB", status.SwapTotalMB)

	if status.CanHibernate {
		pm.log("  * Hibernation Ready  : YES (Swap >= RAM)")
	} else {
		pm.log("  * Hibernation Ready  : NO (Swap %d MB < RAM %d MB)", status.SwapTotalMB, status.RAMTotalMB)
	}

	pm.log("")
	pm.log("Recommended Strategy: %s", status.RecommendedStrategy)
}

// Execute parses the requested mode and executes the power action.
func (pm *PowerManager) Execute(ctx context.Context, mode string) error {
	norm := strings.ToLower(strings.TrimSpace(mode))
	switch norm {
	case "status", "check", "info", "-h", "--help":
		st, err := pm.DetectStatus(ctx)
		if err != nil {
			return err
		}
		pm.PrintStatus(st)
		return nil

	case "lock", "screen", "blank":
		return pm.LockScreen(ctx)

	case "s2idle", "idle":
		return pm.SuspendToIdle(ctx)

	case "deep", "vt", "s3":
		return pm.DeepVTSuspend(ctx)

	case "hibernate", "disk":
		return pm.Hibernate(ctx)

	case "", "auto":
		st, err := pm.DetectStatus(ctx)
		if err != nil {
			return err
		}
		if st.GPUDriver == "nouveau" && st.HasKeplerPStateBug {
			pm.logWarn("Dell M6800 / Nouveau Kepler detected with 'nouveau.pstate=1'.")
			pm.log("  Standard S3 sleep locks up video. Activating safe screen standby & lock...")
			return pm.LockScreen(ctx)
		}
		return pm.LockScreen(ctx)

	default:
		return fmt.Errorf("unknown power mode: %q (supported: lock, s2idle, deep, hibernate, status)", mode)
	}
}

// LockScreen locks the graphical user session and cuts display power using DPMS.
func (pm *PowerManager) LockScreen(ctx context.Context) error {
	pm.log("Entering Safe Screen Standby...")
	pm.logStep("Flushing filesystem buffers to disk...")
	if !pm.dryRun {
		syncFilesystems()
	}

	pm.logStep("Locking desktop session...")
	if !pm.dryRun {
		pm.lockDesktopSession(ctx)
		time.Sleep(300 * time.Millisecond)
	}

	if os.Getenv("DISPLAY") != "" {
		if _, err := exec.LookPath("xset"); err == nil {
			pm.logStep("Powering off display outputs (DPMS force off)...")
			if !pm.dryRun {
				_ = exec.CommandContext(ctx, "xset", "dpms", "force", "off").Run()
			}
		}
	}

	pm.logSuccess("Desktop session locked and monitors placed in low-power standby.")
	pm.log("  Touch any key or move mouse to wake displays instantly.")
	return nil
}

// SuspendToIdle configures kernel s2idle and enters CPU sleep.
func (pm *PowerManager) SuspendToIdle(ctx context.Context) error {
	pm.log("Executing Safe Suspend-to-Idle (s2idle)...")
	memSleepPath := filepath.Join(pm.sysRoot, "power", "mem_sleep")
	data, err := os.ReadFile(memSleepPath)
	if err != nil || !strings.Contains(string(data), "s2idle") {
		pm.logWarn("s2idle is not exposed by current kernel. Falling back to safe screen standby...")
		return pm.LockScreen(ctx)
	}

	pm.logStep("Flushing filesystem caches...")
	if !pm.dryRun {
		syncFilesystems()
	}

	pm.logStep("Locking desktop session...")
	if !pm.dryRun {
		pm.lockDesktopSession(ctx)
	}

	pm.logStep("Configuring mem_sleep to s2idle...")
	if !pm.dryRun {
		if err := os.WriteFile(memSleepPath, []byte("s2idle\n"), 0o644); err != nil {
			// Try via sudo if write fails
			_ = pm.runCommand(ctx, "sudo", "sh", "-c", "echo s2idle > "+memSleepPath)
		}
	}

	pm.logStep("Suspending system to idle state...")
	if !pm.dryRun {
		if err := pm.runCommand(ctx, "systemctl", "--no-ask-password", "suspend"); err != nil {
			_ = pm.runCommand(ctx, "sudo", "systemctl", "--no-ask-password", "suspend")
		}
	}

	pm.logSuccess("System resumed from s2idle.")
	return nil
}

// DeepVTSuspend unbinds DRM master via VT switch before entering ACPI S3 deep sleep.
func (pm *PowerManager) DeepVTSuspend(ctx context.Context) error {
	pm.log("Executing Deep S3 Suspend with Virtual Terminal Switch...")
	st, _ := pm.DetectStatus(ctx)
	origVT := st.ActiveVT
	targetVT := "1"
	if origVT == "1" {
		targetVT = "2"
	}

	pm.logStep("Flushing disk caches...")
	if !pm.dryRun {
		syncFilesystems()
	}

	pm.logStep("Locking desktop session...")
	if !pm.dryRun {
		pm.lockDesktopSession(ctx)
	}

	pm.logStep(fmt.Sprintf("Switching to console tty%s to release DRM master...", targetVT))
	if !pm.dryRun {
		_ = pm.runCommand(ctx, "sudo", "chvt", targetVT)
		time.Sleep(500 * time.Millisecond)
	}

	pm.logStep("Invoking systemctl suspend...")
	if !pm.dryRun {
		if err := pm.runCommand(ctx, "systemctl", "--no-ask-password", "suspend"); err != nil {
			_ = pm.runCommand(ctx, "sudo", "systemctl", "--no-ask-password", "suspend")
		}
		pm.logStep("Resumed from sleep. Waiting for DisplayPort link training...")
		time.Sleep(1 * time.Second)
	}

	pm.logStep(fmt.Sprintf("Restoring graphical session on tty%s...", origVT))
	if !pm.dryRun {
		_ = pm.runCommand(ctx, "sudo", "chvt", origVT)
		time.Sleep(500 * time.Millisecond)
		if os.Getenv("DISPLAY") != "" {
			if _, err := exec.LookPath("xrandr"); err == nil {
				_ = pm.runCommand(ctx, "xrandr", "--auto")
			}
		}
	}

	pm.logSuccess("System resumed successfully from deep sleep.")
	return nil
}

// Hibernate suspends system state to disk swap.
func (pm *PowerManager) Hibernate(ctx context.Context) error {
	pm.log("Checking Hibernation Prerequisites...")
	st, err := pm.DetectStatus(ctx)
	if err != nil {
		return err
	}

	if !st.CanHibernate {
		pm.logError(fmt.Sprintf("Cannot hibernate: swap space (%d MB) is smaller than RAM (%d MB).",
			st.SwapTotalMB, st.RAMTotalMB))
		pm.log("")
		pm.log("To enable hibernation:")
		pm.log("  1. sudo swapoff /swapfile")
		pm.log("  2. sudo dd if=/dev/zero of=/swapfile bs=1G count=%d status=progress", (st.RAMTotalMB/1024)+2)
		pm.log("  3. sudo chmod 600 /swapfile && sudo mkswap /swapfile && sudo swapon /swapfile")
		pm.log("  4. Update /etc/initramfs-tools/conf.d/resume with swap UUID and run update-initramfs -u")
		return fmt.Errorf("insufficient swap space for hibernation (need >= %d MB, have %d MB)",
			st.RAMTotalMB, st.SwapTotalMB)
	}

	pm.logStep("Flushing disks and initiating hibernation...")
	if !pm.dryRun {
		syncFilesystems()
		if err := pm.runCommand(ctx, "systemctl", "--no-ask-password", "hibernate"); err != nil {
			return pm.runCommand(ctx, "sudo", "systemctl", "--no-ask-password", "hibernate")
		}
	}

	pm.logSuccess("System resumed from disk hibernation.")
	return nil
}

func (pm *PowerManager) runCommand(ctx context.Context, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = pm.out
	cmd.Stderr = pm.out
	return cmd.Run()
}

func (pm *PowerManager) lockDesktopSession(ctx context.Context) {
	// Try desktop session lockers in priority order
	lockers := [][]string{
		{"xflock4"},
		{"light-locker-command", "-l"},
		{"loginctl", "lock-session"},
		{"xdg-screensaver", "lock"},
	}

	for _, l := range lockers {
		bin := l[0]
		if _, err := exec.LookPath(bin); err == nil {
			cmd := exec.CommandContext(ctx, bin, l[1:]...)
			_ = cmd.Start()
			return
		}
	}
}
