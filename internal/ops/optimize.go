package ops

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
)

// ErrGuardrailViolation indicates a script was blocked by static security guardrails.
var ErrGuardrailViolation = errors.New("script blocked by security guardrails")

var (
	activeSchedulerRe     = regexp.MustCompile(`\[([^\]]+)\]`)
	blockDeviceBaseRe     = regexp.MustCompile(`^(nvme\d+n\d+|mmcblk\d+|sd[a-z]+|vd[a-z]+|xvd[a-z]+|hd[a-z]+)`)
	supplementaryRootRmRe = regexp.MustCompile(`(?i)\brm\s+(?:[^\n;|&]*\s+)?-(?:[a-zA-Z]*[rf][a-zA-Z]*|-recursive|-force)(?:\s+[^\n;|&]*)?\s+(/|/\*|/boot|/etc|/usr|/var|/home|--no-preserve-root)(?:\s|$)`)
)

// PromptFunc synthesizes an optimization script from a distilled system audit prompt.
type PromptFunc func(ctx context.Context, prompt string) (string, error)

// OptimizerOption configures an Optimizer instance.
type OptimizerOption func(*Optimizer)

// WithPromptFunc attaches an optional LLM synthesis callback to the Optimizer.
func WithPromptFunc(fn PromptFunc) OptimizerOption {
	return func(o *Optimizer) {
		o.PromptFunc = fn
	}
}

// WithOptimizerEnv overrides the environment used for binary discovery and command execution.
func WithOptimizerEnv(env []string) OptimizerOption {
	return func(o *Optimizer) {
		o.env = append([]string{}, env...)
	}
}

// WithFSRoot overrides /proc and /sys roots for deterministic isolated unit testing.
func WithFSRoot(procRoot, sysRoot string) OptimizerOption {
	return func(o *Optimizer) {
		if procRoot != "" {
			o.procRoot = procRoot
		}
		if sysRoot != "" {
			o.sysRoot = sysRoot
		}
	}
}

// DiskMountInfo describes a mounted block filesystem and its kernel I/O scheduler.
type DiskMountInfo struct {
	Device     string `json:"device"`
	MountPoint string `json:"mount_point"`
	FSType     string `json:"fs_type"`
	Scheduler  string `json:"scheduler,omitempty"`
	Rotational bool   `json:"rotational"`
}

// SystemAuditReport holds hardware, kernel, storage, and optional Lynis audit telemetry.
type SystemAuditReport struct {
	OS                string          `json:"os"`
	Distro            string          `json:"distro"`
	Arch              string          `json:"arch"`
	Kernel            string          `json:"kernel"`
	DesktopEnv        string          `json:"desktop_env"`
	DisplayServer     string          `json:"display_server"`
	CPUModel          string          `json:"cpu_model"`
	CPUCores          int             `json:"cpu_cores"`
	CPUGovernor       string          `json:"cpu_governor,omitempty"`
	MemoryTotalMB     int64           `json:"memory_total_mb"`
	MemoryFreeMB      int64           `json:"memory_free_mb"`
	SwapTotalMB       int64           `json:"swap_total_mb"`
	SwapFreeMB        int64           `json:"swap_free_mb"`
	Swappiness        int             `json:"swappiness"`
	InotifyMaxWatches int             `json:"inotify_max_watches"`
	TCPCongestion     string          `json:"tcp_congestion,omitempty"`
	DiskMounts        []DiskMountInfo `json:"disk_mounts"`
	Findings          []string        `json:"findings"`
	LynisRan          bool            `json:"lynis_ran"`
	LynisWarnings     []string        `json:"lynis_warnings"`
	LynisSuggestions  []string        `json:"lynis_suggestions"`
}

// MarkdownSummary formats the SystemAuditReport into a concise high-signal Markdown digest.
func (r *SystemAuditReport) MarkdownSummary() string {
	var sb strings.Builder
	sb.WriteString("# System Hardware, Performance & Security Audit Digest\n\n")
	fmt.Fprintf(&sb, "- Platform: %s (%s) %s, Kernel: %s\n", r.OS, r.Distro, r.Arch, r.Kernel)
	fmt.Fprintf(&sb, "- Desktop / Display: %s / %s\n", r.DesktopEnv, r.DisplayServer)
	fmt.Fprintf(&sb, "- CPU: %s (%d cores, governor=%s)\n", r.CPUModel, r.CPUCores, r.CPUGovernor)
	fmt.Fprintf(&sb, "- Memory: %d MB total, %d MB available\n", r.MemoryTotalMB, r.MemoryFreeMB)
	fmt.Fprintf(&sb, "- Swap: %d MB total, %d MB free (swappiness=%d)\n", r.SwapTotalMB, r.SwapFreeMB, r.Swappiness)
	fmt.Fprintf(&sb, "- Kernel Limits: inotify.max_user_watches=%d, tcp_congestion=%s\n", r.InotifyMaxWatches, r.TCPCongestion)

	if len(r.DiskMounts) > 0 {
		sb.WriteString("\n## Storage Mounts\n")
		for _, m := range r.DiskMounts {
			kind := "SSD/NVMe"
			if m.Rotational {
				kind = "HDD"
			}
			fmt.Fprintf(&sb, "- %s on %s (%s, %s, scheduler=%s)\n", m.Device, m.MountPoint, m.FSType, kind, m.Scheduler)
		}
	}

	if len(r.Findings) > 0 {
		sb.WriteString("\n## Key Resource Findings\n")
		for _, f := range r.Findings {
			fmt.Fprintf(&sb, "- %s\n", f)
		}
	}

	if len(r.LynisWarnings) > 0 {
		sb.WriteString("\n## Lynis Warnings\n")
		for _, w := range r.LynisWarnings {
			fmt.Fprintf(&sb, "- [WARNING] %s\n", w)
		}
	}

	if len(r.LynisSuggestions) > 0 {
		sb.WriteString("\n## Lynis Suggestions\n")
		for _, s := range r.LynisSuggestions {
			fmt.Fprintf(&sb, "- [SUGGESTION] %s\n", s)
		}
	}

	return sb.String()
}

// Optimizer audits host metrics, synthesizes optimization scripts, and enforces security guardrails.
type Optimizer struct {
	plat       *platform.Info
	runner     *security.Runner
	PromptFunc PromptFunc
	env        []string
	procRoot   string
	sysRoot    string
}

// NewOptimizer constructs a native Go system Optimizer.
func NewOptimizer(plat *platform.Info, runner *security.Runner, opts ...OptimizerOption) *Optimizer {
	if runner == nil {
		runner = security.NewRunner(2 * time.Minute)
	}
	o := &Optimizer{
		plat:     plat,
		runner:   runner,
		procRoot: "/proc",
		sysRoot:  "/sys",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	return o
}

// AuditSystem profiles CPU, RAM, swap, disks, kernel parameters, and optional non-root Lynis findings.
func (o *Optimizer) AuditSystem(ctx context.Context, skipLynis bool) (*SystemAuditReport, error) {
	goos := runtime.GOOS
	arch := runtime.GOARCH
	distro := ""
	if o.plat != nil {
		goos = o.plat.OS()
		arch = o.plat.Arch()
		distro = o.plat.Distribution()
	}

	report := &SystemAuditReport{
		OS:                goos,
		Distro:            distro,
		Arch:              arch,
		Swappiness:        -1,
		InotifyMaxWatches: -1,
		DiskMounts:        []DiskMountInfo{},
		Findings:          []string{},
		LynisWarnings:     []string{},
		LynisSuggestions:  []string{},
	}

	report.DesktopEnv = strings.ToLower(strings.TrimSpace(o.getenv("XDG_CURRENT_DESKTOP")))
	if report.DesktopEnv == "" {
		report.DesktopEnv = strings.ToLower(strings.TrimSpace(o.getenv("DESKTOP_SESSION")))
	}
	report.DisplayServer = o.detectDisplayServer(goos)

	report.Kernel = readTrimmedFile(filepath.Join(o.procRoot, "sys/kernel/osrelease"))
	if report.Kernel == "" {
		if unameBin, ok := o.lookupBin("uname"); ok {
			if out, err := o.runCmd(ctx, unameBin, "-r"); err == nil {
				report.Kernel = strings.TrimSpace(out)
			}
		}
	}

	o.populateCPUInfo(ctx, report)
	o.populateMemoryInfo(ctx, report)
	o.populateSysctlInfo(report)
	o.populateDiskMounts(report)
	report.Findings = distillHeuristicFindings(report)

	if !skipLynis {
		o.populateLynisFindings(ctx, report)
	}

	return report, nil
}

// ValidateScript checks a shell script against static anti-destruction guardrails.
func (o *Optimizer) ValidateScript(script string) []security.GuardrailViolation {
	violations := security.ValidateScriptGuardrails(script)
	hasLineGR001 := map[int]bool{}
	for _, v := range violations {
		if v.RuleID == "GR-001" {
			hasLineGR001[v.Line] = true
		}
	}
	for idx, rawLine := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || hasLineGR001[idx+1] {
			continue
		}
		if supplementaryRootRmRe.MatchString(trimmed) {
			violations = append(violations, security.GuardrailViolation{
				RuleID:      "GR-001",
				Description: "recursive deletion targeting root or top-level system directories",
				Line:        idx + 1,
				Snippet:     trimmed,
			})
		}
	}
	return violations
}

// GenerateScript is an alias for SynthesizeScript.
func (o *Optimizer) GenerateScript(ctx context.Context, report *SystemAuditReport) (string, error) {
	return o.SynthesizeScript(ctx, report)
}

// SynthesizeScript generates a tailored optimization script via PromptFunc (or deterministic safe baseline fallback)
// and validates it against security guardrails before returning.
func (o *Optimizer) SynthesizeScript(ctx context.Context, report *SystemAuditReport) (string, error) {
	if report == nil {
		var err error
		report, err = o.AuditSystem(ctx, true)
		if err != nil {
			return "", err
		}
	}

	var script string
	if o.PromptFunc != nil {
		prompt := buildOptimizationPrompt(report)
		rawResp, err := o.PromptFunc(ctx, prompt)
		if err != nil {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			// Offline or unreachable AI backend: fall back to deterministic safe baseline generator.
			script = generateBaselineScript(report)
		} else {
			script = extractShellCodeBlock(rawResp)
		}
	} else {
		script = generateBaselineScript(report)
	}

	script = ensureBashHeader(script)
	if violations := o.ValidateScript(script); len(violations) > 0 {
		return "", formatGuardrailError(violations)
	}
	return script, nil
}

// ExecuteScript validates script guardrails and executes the script safely via bash when dryRun is false.
func (o *Optimizer) ExecuteScript(ctx context.Context, script string, dryRun bool) (string, error) {
	if strings.TrimSpace(script) == "" {
		return "", errors.New("optimization script cannot be empty")
	}

	if violations := o.ValidateScript(script); len(violations) > 0 {
		return "", formatGuardrailError(violations)
	}

	if dryRun {
		return "dry-run: script passed all security guardrails", nil
	}

	bashBin, ok := o.lookupBin("bash")
	if !ok {
		bashBin = "bash"
	}

	tmpDir, err := os.MkdirTemp("", "anx-opt-*")
	if err != nil {
		return "", fmt.Errorf("creating temporary execution dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "optimize.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		return "", fmt.Errorf("writing temporary optimization script: %w", err)
	}

	if _, err := o.runCmd(ctx, bashBin, "-n", scriptPath); err != nil {
		return "", fmt.Errorf("bash syntax validation failed: %w", err)
	}

	return o.runCmd(ctx, bashBin, scriptPath)
}

func (o *Optimizer) populateCPUInfo(ctx context.Context, report *SystemAuditReport) {
	cpuInfoPath := filepath.Join(o.procRoot, "cpuinfo")
	if data, err := os.ReadFile(cpuInfoPath); err == nil {
		cores := 0
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "processor") {
				cores++
			}
			if report.CPUModel == "" && (strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Model")) {
				if idx := strings.IndexByte(line, ':'); idx != -1 {
					report.CPUModel = strings.TrimSpace(line[idx+1:])
				}
			}
		}
		if cores > 0 {
			report.CPUCores = cores
		}
	}

	if report.OS == "darwin" && report.CPUModel == "" {
		if sysctlBin, ok := o.lookupBin("sysctl"); ok {
			if out, err := o.runCmd(ctx, sysctlBin, "-n", "machdep.cpu.brand_string"); err == nil {
				report.CPUModel = strings.TrimSpace(out)
			}
		}
	}

	if report.CPUModel == "" {
		report.CPUModel = "unknown"
	}
	if report.CPUCores <= 0 {
		report.CPUCores = runtime.NumCPU()
	}

	report.CPUGovernor = readTrimmedFile(filepath.Join(o.sysRoot, "devices/system/cpu/cpu0/cpufreq/scaling_governor"))
}

func (o *Optimizer) populateMemoryInfo(ctx context.Context, report *SystemAuditReport) {
	memPath := filepath.Join(o.procRoot, "meminfo")
	if data, err := os.ReadFile(memPath); err == nil {
		var memFreeKB, memAvailKB int64
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 2 {
				continue
			}
			val, parseErr := strconv.ParseInt(fields[1], 10, 64)
			if parseErr != nil {
				continue
			}
			switch strings.TrimSuffix(fields[0], ":") {
			case "MemTotal":
				report.MemoryTotalMB = val / 1024
			case "MemAvailable":
				memAvailKB = val
			case "MemFree":
				memFreeKB = val
			case "SwapTotal":
				report.SwapTotalMB = val / 1024
			case "SwapFree":
				report.SwapFreeMB = val / 1024
			}
		}
		if memAvailKB > 0 {
			report.MemoryFreeMB = memAvailKB / 1024
		} else {
			report.MemoryFreeMB = memFreeKB / 1024
		}
		return
	}

	if report.OS == "darwin" {
		if sysctlBin, ok := o.lookupBin("sysctl"); ok {
			if out, err := o.runCmd(ctx, sysctlBin, "-n", "hw.memsize"); err == nil {
				if bytesVal, pErr := strconv.ParseInt(strings.TrimSpace(out), 10, 64); pErr == nil {
					report.MemoryTotalMB = bytesVal / (1024 * 1024)
				}
			}
		}
	}
}

func (o *Optimizer) populateSysctlInfo(report *SystemAuditReport) {
	if raw := readTrimmedFile(filepath.Join(o.procRoot, "sys/vm/swappiness")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			report.Swappiness = v
		}
	}
	if raw := readTrimmedFile(filepath.Join(o.procRoot, "sys/fs/inotify/max_user_watches")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			report.InotifyMaxWatches = v
		}
	}
	report.TCPCongestion = readTrimmedFile(filepath.Join(o.procRoot, "sys/net/ipv4/tcp_congestion_control"))
}

func (o *Optimizer) populateDiskMounts(report *SystemAuditReport) {
	mountsPath := filepath.Join(o.procRoot, "mounts")
	data, err := os.ReadFile(mountsPath)
	if err != nil {
		return
	}

	seenDevices := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		dev := fields[0]
		mountPt := fields[1]
		fsType := fields[2]

		if !strings.HasPrefix(dev, "/dev/") || strings.HasPrefix(dev, "/dev/loop") {
			continue
		}
		if seenDevices[dev] {
			continue
		}
		seenDevices[dev] = true

		info := DiskMountInfo{
			Device:     dev,
			MountPoint: mountPt,
			FSType:     fsType,
		}

		baseBlock := baseBlockDeviceName(filepath.Base(dev))
		if baseBlock != "" {
			rawSched := readTrimmedFile(filepath.Join(o.sysRoot, "block", baseBlock, "queue/scheduler"))
			if matches := activeSchedulerRe.FindStringSubmatch(rawSched); len(matches) == 2 {
				info.Scheduler = matches[1]
			} else if rawSched != "" {
				info.Scheduler = rawSched
			}

			rawRota := readTrimmedFile(filepath.Join(o.sysRoot, "block", baseBlock, "queue/rotational"))
			info.Rotational = (rawRota == "1")
		}

		report.DiskMounts = append(report.DiskMounts, info)
	}
}

func (o *Optimizer) populateLynisFindings(ctx context.Context, report *SystemAuditReport) {
	lynisBin, ok := o.lookupBin("lynis")
	if !ok {
		return
	}

	tmpDir, err := os.MkdirTemp("", "anx-lynis-*")
	if err != nil {
		return
	}
	defer os.RemoveAll(tmpDir)

	reportFile := filepath.Join(tmpDir, "lynis-report.dat")
	logFile := filepath.Join(tmpDir, "lynis.log")

	out, _ := o.runner.RunWithOptions(
		ctx,
		tmpDir,
		o.effectiveEnv(),
		lynisBin,
		"audit",
		"system",
		"--quick",
		"--no-colors",
		"--no-plugins",
		"--report-file",
		reportFile,
		"--log-file",
		logFile,
	)

	report.LynisRan = true
	warnSet := map[string]bool{}
	sugSet := map[string]bool{}

	if datBytes, readErr := os.ReadFile(reportFile); readErr == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(datBytes)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			switch {
			case strings.HasPrefix(line, "warning[]="):
				val := cleanLynisEntry(strings.TrimPrefix(line, "warning[]="))
				if val != "" && !warnSet[val] {
					warnSet[val] = true
					report.LynisWarnings = append(report.LynisWarnings, val)
				}
			case strings.HasPrefix(line, "suggestion[]="):
				val := cleanLynisEntry(strings.TrimPrefix(line, "suggestion[]="))
				if val != "" && !sugSet[val] && len(report.LynisSuggestions) < 12 {
					sugSet[val] = true
					report.LynisSuggestions = append(report.LynisSuggestions, val)
				}
			}
		}
	}

	// Fallback stdout parsing if lynis emitted lines directly without writing reportFile.
	if len(report.LynisWarnings) == 0 && len(report.LynisSuggestions) == 0 && out != "" {
		scanner := bufio.NewScanner(strings.NewReader(out))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			lower := strings.ToLower(line)
			switch {
			case strings.Contains(lower, "warning"):
				if !warnSet[line] {
					warnSet[line] = true
					report.LynisWarnings = append(report.LynisWarnings, line)
				}
			case strings.Contains(lower, "suggestion"):
				if !sugSet[line] && len(report.LynisSuggestions) < 12 {
					sugSet[line] = true
					report.LynisSuggestions = append(report.LynisSuggestions, line)
				}
			}
		}
	}
}

func (o *Optimizer) detectDisplayServer(goos string) string {
	if st := strings.ToLower(strings.TrimSpace(o.getenv("XDG_SESSION_TYPE"))); st == "wayland" || st == "x11" {
		return st
	}
	if o.getenv("WAYLAND_DISPLAY") != "" {
		return "wayland"
	}
	if o.getenv("DISPLAY") != "" {
		return "x11"
	}
	if goos == "darwin" {
		return "quartz"
	}
	return "headless"
}

func (o *Optimizer) effectiveEnv() []string {
	if len(o.env) > 0 {
		return o.env
	}
	return os.Environ()
}

func (o *Optimizer) getenv(key string) string {
	if len(o.env) > 0 {
		return lookupEnvValue(o.env, key)
	}
	return os.Getenv(key)
}

func (o *Optimizer) lookupBin(name string) (string, bool) {
	return lookupBinaryInEnv(name, o.effectiveEnv())
}

func (o *Optimizer) runCmd(ctx context.Context, binary string, args ...string) (string, error) {
	return o.runner.RunWithOptions(ctx, "", o.effectiveEnv(), binary, args...)
}

func distillHeuristicFindings(report *SystemAuditReport) []string {
	findings := []string{}
	if report.Swappiness > 20 {
		findings = append(findings, fmt.Sprintf("vm.swappiness is %d; lowering to 15 reduces premature SSD swap thrashing", report.Swappiness))
	}
	if report.InotifyMaxWatches > 0 && report.InotifyMaxWatches < 524288 {
		findings = append(findings, fmt.Sprintf("fs.inotify.max_user_watches is %d; raising to 524288 prevents ENOSPC in Neovim, LSPs, and file watchers", report.InotifyMaxWatches))
	}
	if report.TCPCongestion != "" && report.TCPCongestion != "bbr" {
		findings = append(findings, fmt.Sprintf("TCP congestion control is %q; BBR improves throughput and reduces bufferbloat where supported", report.TCPCongestion))
	}
	if report.MemoryTotalMB > 0 && report.MemoryFreeMB*10 < report.MemoryTotalMB {
		findings = append(findings, fmt.Sprintf("available RAM is below 10%% (%d MB / %d MB)", report.MemoryFreeMB, report.MemoryTotalMB))
	}
	return findings
}

func buildOptimizationPrompt(report *SystemAuditReport) string {
	return "Generate a safe, idempotent, non-destructive POSIX/Bash workstation optimization script " +
		"using /etc/sysctl.d drop-in files with # WHY comments for every tuned setting.\n\n" +
		report.MarkdownSummary()
}

func generateBaselineScript(report *SystemAuditReport) string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("set -euo pipefail\n\n")
	sb.WriteString("# Deterministic Baseline Workstation Optimization Script (anx)\n")
	fmt.Fprintf(&sb, "# Target Platform: %s (%s) / %s\n\n", report.OS, report.Distro, report.CPUModel)

	if report.OS != "linux" {
		sb.WriteString("echo \"[anx-optimize] No baseline Linux sysctl drop-ins required for this OS.\"\n")
		return sb.String()
	}

	sb.WriteString("SYSCTL_DROPIN=\"/etc/sysctl.d/99-anx-workstation-tuning.conf\"\n")
	sb.WriteString("if [[ \"$(id -u)\" -ne 0 ]]; then\n")
	sb.WriteString("  echo \"[anx-optimize] Run as root (sudo) to write ${SYSCTL_DROPIN}.\" >&2\n")
	sb.WriteString("  exit 0\n")
	sb.WriteString("fi\n\n")
	sb.WriteString("if [[ -f \"${SYSCTL_DROPIN}\" ]]; then\n")
	sb.WriteString("  cp -p \"${SYSCTL_DROPIN}\" \"${SYSCTL_DROPIN}.bak.$(date +%Y%m%d%H%M%S)\"\n")
	sb.WriteString("fi\n\n")
	sb.WriteString("cat <<'EOF' > \"${SYSCTL_DROPIN}\"\n")
	sb.WriteString("# Managed by anx optimize baseline generator\n")
	sb.WriteString("# WHY: Reduce premature page swapping on SSD/NVMe developer workstations\n")
	sb.WriteString("vm.swappiness = 15\n\n")
	sb.WriteString("# WHY: Prevent ENOSPC errors in Neovim LSP, Go/Node watchers, and large monorepos\n")
	sb.WriteString("fs.inotify.max_user_watches = 524288\n")
	sb.WriteString("fs.inotify.max_user_instances = 1024\n")
	sb.WriteString("EOF\n\n")
	sb.WriteString("sysctl --system >/dev/null 2>&1 || true\n")
	sb.WriteString("echo \"[anx-optimize] Applied workstation sysctl drop-in cleanly.\"\n")
	return sb.String()
}

func extractShellCodeBlock(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if idx := strings.Index(trimmed, "```bash"); idx != -1 {
		rest := trimmed[idx+len("```bash"):]
		if end := strings.Index(rest, "```"); end != -1 {
			return strings.TrimSpace(rest[:end]) + "\n"
		}
	}
	if idx := strings.Index(trimmed, "```sh"); idx != -1 {
		rest := trimmed[idx+len("```sh"):]
		if end := strings.Index(rest, "```"); end != -1 {
			return strings.TrimSpace(rest[:end]) + "\n"
		}
	}
	if idx := strings.Index(trimmed, "```"); idx != -1 {
		rest := trimmed[idx+len("```"):]
		if end := strings.Index(rest, "```"); end != -1 {
			return strings.TrimSpace(rest[:end]) + "\n"
		}
	}
	return trimmed + "\n"
}

func ensureBashHeader(script string) string {
	trimmed := strings.TrimSpace(script)
	if strings.HasPrefix(trimmed, "#!") {
		return trimmed + "\n"
	}
	return "#!/usr/bin/env bash\nset -euo pipefail\n\n" + trimmed + "\n"
}

func formatGuardrailError(violations []security.GuardrailViolation) error {
	parts := make([]string, 0, len(violations))
	for _, v := range violations {
		parts = append(parts, fmt.Sprintf("[%s line %d: %s]", v.RuleID, v.Line, v.Description))
	}
	return fmt.Errorf("%w: %s", ErrGuardrailViolation, strings.Join(parts, "; "))
}

func cleanLynisEntry(raw string) string {
	parts := strings.Split(raw, "|")
	if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
		return strings.TrimSpace(parts[0]) + ": " + strings.TrimSpace(parts[1])
	}
	return strings.Trim(raw, "| ")
}

func baseBlockDeviceName(devBase string) string {
	if matches := blockDeviceBaseRe.FindStringSubmatch(devBase); len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func readTrimmedFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
