package hacks

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
)

// ProbeOption configures hardware probing roots and environment overrides.
type ProbeOption func(*probeConfig)

type probeConfig struct {
	sysRoot  string
	procRoot string
	env      []string
	hasEnv   bool
}

// WithProbeRoots overrides /sys and /proc roots for deterministic unit testing.
func WithProbeRoots(sysRoot, procRoot string) ProbeOption {
	return func(c *probeConfig) {
		if sysRoot != "" {
			c.sysRoot = sysRoot
		}
		if procRoot != "" {
			c.procRoot = procRoot
		}
	}
}

// WithProbeEnv overrides environment variable lookups during hardware probing.
func WithProbeEnv(env []string) ProbeOption {
	return func(c *probeConfig) {
		c.env = append([]string{}, env...)
		c.hasEnv = true
	}
}

// ProbeHardware inspects host DMI, CPU, RAM, GPU, desktop environment, and display server safely
// without failing in minimal containers or headless hosts.
func ProbeHardware(
	ctx context.Context,
	plat *platform.Info,
	runner *security.Runner,
	opts ...ProbeOption,
) domain.HardwareProfile {
	cfg := &probeConfig{
		sysRoot:  "/sys",
		procRoot: "/proc",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	goos := runtime.GOOS
	arch := runtime.GOARCH
	distro := ""
	if plat != nil {
		goos = plat.OS()
		arch = plat.Arch()
		distro = plat.Distribution()
	}

	hw := domain.HardwareProfile{
		OS:         goos,
		Arch:       arch,
		Distro:     distro,
		GPUModels:  []string{},
		GPUDrivers: []string{},
	}

	hw.DMIVendor = readFirstTrimmed(
		filepath.Join(cfg.sysRoot, "class/dmi/id/sys_vendor"),
		filepath.Join(cfg.sysRoot, "devices/virtual/dmi/id/sys_vendor"),
	)
	hw.DMIProduct = readFirstTrimmed(
		filepath.Join(cfg.sysRoot, "class/dmi/id/product_name"),
		filepath.Join(cfg.sysRoot, "devices/virtual/dmi/id/product_name"),
	)

	hw.Kernel = readFirstTrimmed(filepath.Join(cfg.procRoot, "sys/kernel/osrelease"))
	if hw.Kernel == "" && runner != nil {
		if out, err := runner.Run(ctx, "uname", "-r"); err == nil {
			hw.Kernel = strings.TrimSpace(out)
		}
	}

	hw.CPUModel, hw.CPUCores = probeCPU(ctx, goos, cfg.procRoot, runner)
	hw.MemoryTotalMB = probeMemoryMB(ctx, goos, cfg.procRoot, runner)
	hw.DesktopEnv = probeDesktopEnv(cfg)
	hw.DisplayServer = probeDisplayServer(goos, cfg)
	hw.GPUModels, hw.GPUDrivers = probeGPUs(ctx, goos, cfg.sysRoot, runner)

	return hw
}

// MatchHacks returns all HackProfiles whose hardware predicates match hw.
func MatchHacks(hw domain.HardwareProfile, profiles []domain.HackProfile) []domain.HackProfile {
	matched := []domain.HackProfile{}
	for _, p := range profiles {
		if matchesHardware(hw, p.Match) {
			matched = append(matched, p)
		}
	}
	return matched
}

func matchesHardware(hw domain.HardwareProfile, m domain.HardwareMatcher) bool {
	hasPredicate := len(m.OS) > 0 ||
		len(m.Distros) > 0 ||
		m.DMIProduct != "" ||
		m.DMIVendor != "" ||
		m.GPUVendor != "" ||
		m.GPUDriver != "" ||
		m.GPUModel != "" ||
		len(m.DesktopEnv) > 0 ||
		len(m.DisplayServer) > 0

	if !hasPredicate {
		return false
	}

	if len(m.OS) > 0 && !containsFold(m.OS, hw.OS) {
		return false
	}
	if len(m.Distros) > 0 && !containsFold(m.Distros, hw.Distro) {
		return false
	}
	if m.DMIVendor != "" && !containsSubstringFold(hw.DMIVendor, m.DMIVendor) {
		return false
	}
	if m.DMIProduct != "" && !containsSubstringFold(hw.DMIProduct, m.DMIProduct) {
		return false
	}
	if m.GPUDriver != "" && !anyContainsFold(hw.GPUDrivers, m.GPUDriver) {
		return false
	}
	if m.GPUModel != "" && !anyContainsFold(hw.GPUModels, m.GPUModel) {
		return false
	}
	if m.GPUVendor != "" && !anyContainsFold(hw.GPUModels, m.GPUVendor) && !anyContainsFold(hw.GPUDrivers, m.GPUVendor) {
		return false
	}
	if len(m.DesktopEnv) > 0 && !matchesDesktopEnv(hw.DesktopEnv, m.DesktopEnv) {
		return false
	}
	if len(m.DisplayServer) > 0 && !containsFold(m.DisplayServer, hw.DisplayServer) {
		return false
	}

	return true
}

func matchesDesktopEnv(actual string, allowed []string) bool {
	if actual == "" {
		return false
	}
	// XDG_CURRENT_DESKTOP can be colon-separated such as "ubuntu:GNOME" or "X-Cinnamon".
	tokens := strings.FieldsFunc(strings.ToLower(actual), func(r rune) bool {
		return r == ':' || r == ';' || r == ','
	})
	for _, candidate := range allowed {
		want := strings.ToLower(strings.TrimSpace(candidate))
		if want == "" {
			continue
		}
		if slices.Contains(tokens, want) || strings.Contains(strings.ToLower(actual), want) {
			return true
		}
	}
	return false
}

func probeCPU(ctx context.Context, goos, procRoot string, runner *security.Runner) (string, int) {
	var model string
	cores := 0

	if data, err := os.ReadFile(filepath.Join(procRoot, "cpuinfo")); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "processor") {
				cores++
			}
			if model == "" && (strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Model")) {
				if idx := strings.IndexByte(line, ':'); idx != -1 {
					model = strings.TrimSpace(line[idx+1:])
				}
			}
		}
	}

	if goos == "darwin" && model == "" && runner != nil {
		if out, err := runner.Run(ctx, "sysctl", "-n", "machdep.cpu.brand_string"); err == nil {
			model = strings.TrimSpace(out)
		}
	}

	if cores <= 0 {
		cores = runtime.NumCPU()
	}
	return model, cores
}

func probeMemoryMB(ctx context.Context, goos, procRoot string, runner *security.Runner) int64 {
	if data, err := os.ReadFile(filepath.Join(procRoot, "meminfo")); err == nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 2 && fields[0] == "MemTotal:" {
				if kb, pErr := strconv.ParseInt(fields[1], 10, 64); pErr == nil {
					return kb / 1024
				}
			}
		}
	}

	if goos == "darwin" && runner != nil {
		if out, err := runner.Run(ctx, "sysctl", "-n", "hw.memsize"); err == nil {
			if b, pErr := strconv.ParseInt(strings.TrimSpace(out), 10, 64); pErr == nil {
				return b / (1024 * 1024)
			}
		}
	}
	return 0
}

func probeDesktopEnv(cfg *probeConfig) string {
	de := strings.TrimSpace(cfg.getenv("XDG_CURRENT_DESKTOP"))
	if de == "" {
		de = strings.TrimSpace(cfg.getenv("DESKTOP_SESSION"))
	}
	return strings.ToLower(de)
}

func probeDisplayServer(goos string, cfg *probeConfig) string {
	sessionType := strings.ToLower(strings.TrimSpace(cfg.getenv("XDG_SESSION_TYPE")))
	if sessionType == "wayland" || sessionType == "x11" {
		return sessionType
	}
	if cfg.getenv("WAYLAND_DISPLAY") != "" {
		return "wayland"
	}
	if cfg.getenv("DISPLAY") != "" {
		return "x11"
	}
	if goos == "darwin" {
		return "quartz"
	}
	return "headless"
}

func probeGPUs(ctx context.Context, goos, sysRoot string, runner *security.Runner) ([]string, []string) {
	models := []string{}
	drivers := []string{}
	seenModels := map[string]bool{}
	seenDrivers := map[string]bool{}

	pciDevicesDir := filepath.Join(sysRoot, "bus/pci/devices")
	if entries, err := os.ReadDir(pciDevicesDir); err == nil {
		for _, entry := range entries {
			devDir := filepath.Join(pciDevicesDir, entry.Name())
			pciClass := strings.ToLower(readFirstTrimmed(filepath.Join(devDir, "class")))
			uevent := readFirstTrimmed(filepath.Join(devDir, "uevent"))

			isDisplay := strings.HasPrefix(pciClass, "0x03") ||
				strings.Contains(strings.ToLower(uevent), "pci_class=300") ||
				strings.Contains(strings.ToLower(uevent), "pci_class=302") ||
				strings.Contains(strings.ToLower(uevent), "pci_class=380")
			if !isDisplay {
				continue
			}

			var driver, pciID string
			scanner := bufio.NewScanner(strings.NewReader(uevent))
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				switch {
				case strings.HasPrefix(line, "DRIVER="):
					driver = strings.TrimPrefix(line, "DRIVER=")
				case strings.HasPrefix(line, "PCI_ID="):
					pciID = strings.TrimPrefix(line, "PCI_ID=")
				}
			}

			if driver == "" {
				if linkTarget, lErr := os.Readlink(filepath.Join(devDir, "driver")); lErr == nil {
					driver = filepath.Base(linkTarget)
				}
			}
			if driver != "" && !seenDrivers[driver] {
				seenDrivers[driver] = true
				drivers = append(drivers, driver)
			}

			if pciID != "" {
				vendorModel := mapPCIVendorID(pciID)
				if !seenModels[vendorModel] {
					seenModels[vendorModel] = true
					models = append(models, vendorModel)
				}
			}
		}
	}

	if runner != nil && sysRoot == "/sys" {
		switch goos {
		case "linux":
			if _, err := exec.LookPath("lspci"); err == nil {
				if out, rErr := runner.Run(ctx, "lspci"); rErr == nil {
					enrichModelsFromLspci(out, &models, seenModels)
				}
			}
		case "darwin":
			if _, err := exec.LookPath("system_profiler"); err == nil {
				if out, rErr := runner.Run(ctx, "system_profiler", "SPDisplaysDataType"); rErr == nil {
					enrichModelsFromSystemProfiler(out, &models, seenModels)
				}
			}
		}
	}

	return models, drivers
}

func enrichModelsFromLspci(out string, models *[]string, seen map[string]bool) {
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "vga") && !strings.Contains(lower, "3d controller") && !strings.Contains(lower, "display controller") {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		desc := strings.TrimSpace(line)
		if len(parts) == 2 {
			desc = strings.TrimSpace(parts[1])
		}
		if desc != "" && !seen[desc] {
			seen[desc] = true
			*models = append(*models, desc)
		}
	}
}

func enrichModelsFromSystemProfiler(out string, models *[]string, seen map[string]bool) {
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Chipset Model:") {
			desc := strings.TrimSpace(strings.TrimPrefix(line, "Chipset Model:"))
			if desc != "" && !seen[desc] {
				seen[desc] = true
				*models = append(*models, desc)
			}
		}
	}
}

func mapPCIVendorID(pciID string) string {
	upper := strings.ToUpper(pciID)
	switch {
	case strings.HasPrefix(upper, "10DE:"):
		return "NVIDIA (" + upper + ")"
	case strings.HasPrefix(upper, "8086:"):
		return "Intel (" + upper + ")"
	case strings.HasPrefix(upper, "1002:"), strings.HasPrefix(upper, "1022:"):
		return "AMD (" + upper + ")"
	default:
		return "PCI GPU (" + upper + ")"
	}
}

func (c *probeConfig) getenv(key string) string {
	if c.hasEnv {
		prefix := key + "="
		for _, kv := range c.env {
			if strings.HasPrefix(kv, prefix) {
				return strings.TrimPrefix(kv, prefix)
			}
		}
		return ""
	}
	return os.Getenv(key)
}

func readFirstTrimmed(paths ...string) string {
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			if trimmed := strings.TrimSpace(string(data)); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func containsFold(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func containsSubstringFold(haystack, needle string) bool {
	if haystack == "" || needle == "" {
		return false
	}
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}

func anyContainsFold(items []string, needle string) bool {
	for _, item := range items {
		if containsSubstringFold(item, needle) {
			return true
		}
	}
	return false
}
