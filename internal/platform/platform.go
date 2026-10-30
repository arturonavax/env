package platform

import (
	"bufio"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

// Info holds discovered platform, OS, architecture, package manager, and directory details.
type Info struct {
	osType          string
	archType        string
	distribution    string
	distroFamily    string
	packageManagers []string
	isWSL           bool
	isTermux        bool
	isRoot          bool
	hasSudo         bool
	homeDir         string
	configDir       string
	stateDir        string
	cacheDir        string
	localBinDir     string
	toolsBinDir     string
	desktopEnv      string
	displayServer   string
	currentShell    string
	currentTerminal string
}

// TestInfoParams specifies overrides for deterministic unit testing of platform-dependent services.
type TestInfoParams struct {
	OS              string
	Arch            string
	Distribution    string
	DistroFamily    string
	PackageManagers []string
	IsWSL           bool
	IsTermux        bool
	IsRoot          bool
	HasSudo         bool
	HomeDir         string
	ConfigDir       string
	StateDir        string
	CacheDir        string
	LocalBinDir     string
	ToolsBinDir     string
	DesktopEnv      string
	DisplayServer   string
	CurrentShell    string
	CurrentTerminal string
}

// NewTestInfo builds an isolated Info without probing host filesystems.
func NewTestInfo(p TestInfoParams) *Info {
	goos := p.OS
	if goos == "" {
		goos = "linux"
	}
	arch := p.Arch
	if arch == "" {
		arch = "amd64"
	}
	home := p.HomeDir
	cfgDir := p.ConfigDir
	if cfgDir == "" && home != "" {
		cfgDir = filepath.Join(home, ".config", "anx")
	}
	stateDir := p.StateDir
	if stateDir == "" && home != "" {
		stateDir = filepath.Join(home, ".local", "state", "anx")
	}
	cacheDir := p.CacheDir
	if cacheDir == "" && home != "" {
		cacheDir = filepath.Join(home, ".cache", "anx")
	}
	binDir := p.LocalBinDir
	if binDir == "" && home != "" {
		binDir = filepath.Join(home, ".local", "bin")
	}
	pms := p.PackageManagers
	if pms == nil {
		pms = []string{}
	}
	family := p.DistroFamily
	if family == "" {
		family = inferDistroFamily(goos, p.Distribution, p.IsTermux)
	}

	return &Info{
		osType:          goos,
		archType:        arch,
		distribution:    p.Distribution,
		distroFamily:    family,
		packageManagers: pms,
		isWSL:           p.IsWSL,
		isTermux:        p.IsTermux,
		isRoot:          p.IsRoot,
		hasSudo:         p.HasSudo,
		homeDir:         home,
		configDir:       cfgDir,
		stateDir:        stateDir,
		cacheDir:        cacheDir,
		localBinDir:     binDir,
		toolsBinDir:     p.ToolsBinDir,
		desktopEnv:      p.DesktopEnv,
		displayServer:   p.DisplayServer,
		currentShell:    p.CurrentShell,
		currentTerminal: p.CurrentTerminal,
	}
}

func (p *Info) OS() string                { return p.osType }
func (p *Info) Arch() string              { return p.archType }
func (p *Info) Distribution() string      { return p.distribution }
func (p *Info) DistroFamily() string      { return p.distroFamily }
func (p *Info) PackageManagers() []string { return append([]string{}, p.packageManagers...) }
func (p *Info) PrimaryPackageManager() string {
	if len(p.packageManagers) == 0 {
		return ""
	}
	return p.packageManagers[0]
}
func (p *Info) IsWSL() bool             { return p.isWSL }
func (p *Info) IsTermux() bool          { return p.isTermux }
func (p *Info) IsRoot() bool            { return p.isRoot }
func (p *Info) HasSudo() bool           { return p.hasSudo }
func (p *Info) HomeDir() string         { return p.homeDir }
func (p *Info) ConfigDir() string       { return p.configDir }
func (p *Info) StateDir() string        { return p.stateDir }
func (p *Info) CacheDir() string        { return p.cacheDir }
func (p *Info) LocalBinDir() string     { return p.localBinDir }
func (p *Info) ToolsBinDir() string     { return p.toolsBinDir }
func (p *Info) DesktopEnv() string      { return p.desktopEnv }
func (p *Info) DisplayServer() string   { return p.displayServer }
func (p *Info) CurrentShell() string    { return p.currentShell }
func (p *Info) CurrentTerminal() string { return p.currentTerminal }

// ExpandPath expands leading "~/" or "$HOME/" against Info.HomeDir().
func (p *Info) ExpandPath(raw string) string {
	if raw == "" {
		return ""
	}
	if raw == "~" {
		return p.homeDir
	}
	if strings.HasPrefix(raw, "~/") || strings.HasPrefix(raw, "~\\") {
		return filepath.Join(p.homeDir, raw[2:])
	}
	if strings.HasPrefix(raw, "$HOME/") {
		return filepath.Join(p.homeDir, strings.TrimPrefix(raw, "$HOME/"))
	}
	return raw
}

// Detect performs system environment inspection and returns platform Info.
func Detect() *Info {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	info := &Info{
		osType:          goos,
		archType:        goarch,
		packageManagers: []string{},
	}

	home, err := os.UserHomeDir()
	if err != nil {
		if u, uErr := user.Current(); uErr == nil {
			home = u.HomeDir
		} else {
			home = os.Getenv("HOME")
		}
	}
	info.homeDir = home
	info.isRoot = (os.Geteuid() == 0)

	if _, err := exec.LookPath("sudo"); err == nil {
		info.hasSudo = true
	}

	prefix := os.Getenv("PREFIX")
	if strings.Contains(prefix, "com.termux") || fileExists("/data/data/com.termux/files") {
		info.isTermux = true
	}

	if goos == "linux" {
		info.distribution = detectLinuxDistro()
		info.isWSL = detectWSL()
	}

	info.distroFamily = inferDistroFamily(goos, info.distribution, info.isTermux)
	info.packageManagers = detectPackageManagers(goos, info.isTermux)

	switch goos {
	case "windows":
		appData := os.Getenv("APPDATA")
		localAppData := os.Getenv("LOCALAPPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		info.configDir = filepath.Join(appData, "anx")
		info.stateDir = filepath.Join(localAppData, "anx")
		info.cacheDir = filepath.Join(localAppData, "anx", "cache")
		info.localBinDir = filepath.Join(localAppData, "Programs", "anx", "bin")
	case "darwin":
		info.configDir = filepath.Join(home, ".config", "anx")
		info.stateDir = filepath.Join(home, "Library", "Application Support", "anx")
		info.cacheDir = filepath.Join(home, "Library", "Caches", "anx")
		info.localBinDir = filepath.Join(home, ".local", "bin")
	default:
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" || info.isTermux {
			xdgConfig = filepath.Join(home, ".config")
		}
		xdgState := os.Getenv("XDG_STATE_HOME")
		if xdgState == "" || info.isTermux {
			xdgState = filepath.Join(home, ".local", "state")
		}
		xdgCache := os.Getenv("XDG_CACHE_HOME")
		if xdgCache == "" || info.isTermux {
			xdgCache = filepath.Join(home, ".cache")
		}
		info.configDir = filepath.Join(xdgConfig, "anx")
		info.stateDir = filepath.Join(xdgState, "anx")
		info.cacheDir = filepath.Join(xdgCache, "anx")
		info.localBinDir = filepath.Join(home, ".local", "bin")
	}

	customToolsBin := os.Getenv("ANX_TOOLS_BIN")
	if customToolsBin != "" {
		info.toolsBinDir = customToolsBin
	} else if goos == "windows" {
		info.toolsBinDir = filepath.Join(info.stateDir, "tools", "bin")
	} else {
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData == "" || info.isTermux {
			xdgData = filepath.Join(home, ".local", "share")
		}
		info.toolsBinDir = filepath.Join(xdgData, "anx", "bin")
	}

	info.desktopEnv = detectDesktopEnv(goos)
	info.displayServer = detectDisplayServer(goos)
	info.currentShell = detectShell(goos)
	info.currentTerminal = detectTerminal()

	return info
}

func detectDesktopEnv(goos string) string {
	if goos == "darwin" {
		return "macos"
	}
	if goos == "windows" {
		return "windows"
	}
	raw := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	if raw == "" {
		raw = strings.ToLower(os.Getenv("DESKTOP_SESSION"))
	}
	switch {
	case strings.Contains(raw, "gnome") || strings.Contains(raw, "unity"):
		return "gnome"
	case strings.Contains(raw, "xfce"):
		return "xfce"
	case strings.Contains(raw, "hyprland"):
		return "hyprland"
	case strings.Contains(raw, "sway"):
		return "sway"
	case strings.Contains(raw, "kde") || strings.Contains(raw, "plasma"):
		return "kde"
	case raw != "":
		return raw
	default:
		return "headless"
	}
}

func detectDisplayServer(goos string) string {
	if goos == "darwin" {
		return "quartz"
	}
	if goos == "windows" {
		return "win32"
	}
	sess := strings.ToLower(os.Getenv("XDG_SESSION_TYPE"))
	if sess == "wayland" || os.Getenv("WAYLAND_DISPLAY") != "" {
		return "wayland"
	}
	if sess == "x11" || os.Getenv("DISPLAY") != "" {
		return "x11"
	}
	return "headless"
}

func detectShell(goos string) string {
	raw := os.Getenv("SHELL")
	if raw != "" {
		base := strings.ToLower(filepath.Base(raw))
		for _, s := range []string{"zsh", "bash", "fish", "nu"} {
			if strings.Contains(base, s) {
				return s
			}
		}
	}
	if goos == "windows" {
		return "powershell"
	}
	return "zsh"
}

func detectTerminal() string {
	if os.Getenv("WT_SESSION") != "" {
		return "windows-terminal"
	}
	if os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return "ghostty"
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return "kitty"
	}
	if os.Getenv("WEZTERM_PANE") != "" {
		return "wezterm"
	}
	if os.Getenv("ALACRITTY_LOG") != "" || os.Getenv("ALACRITTY_WINDOW_ID") != "" {
		return "alacritty"
	}
	if os.Getenv("WARP_IS_LOCAL_SHELL_SESSION") != "" {
		return "warp"
	}

	tp := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	switch {
	case strings.Contains(tp, "ghostty"):
		return "ghostty"
	case strings.Contains(tp, "kitty"):
		return "kitty"
	case strings.Contains(tp, "wezterm"):
		return "wezterm"
	case strings.Contains(tp, "alacritty"):
		return "alacritty"
	case strings.Contains(tp, "warp"):
		return "warp"
	case strings.Contains(tp, "windowsterminal"):
		return "windows-terminal"
	default:
		return "unknown"
	}
}

func inferDistroFamily(goos, distro string, isTermux bool) string {
	if isTermux {
		return "termux"
	}
	switch goos {
	case "darwin":
		return "macos"
	case "windows":
		return "windows"
	}
	switch distro {
	case "ubuntu", "debian", "pop", "linuxmint", "kali", "raspbian", "elementary", "zorin":
		return "debian"
	case "fedora", "rhel", "centos", "rocky", "almalinux", "amzn":
		return "rhel"
	case "arch", "manjaro", "endeavouros", "garuda", "cachyos":
		return "arch"
	case "opensuse", "opensuse-leap", "opensuse-tumbleweed", "sles":
		return "suse"
	case "alpine":
		return "alpine"
	default:
		return distro
	}
}

func detectPackageManagers(goos string, isTermux bool) []string {
	var candidates []string
	switch {
	case isTermux:
		candidates = []string{"pkg", "apt"}
	case goos == "darwin":
		candidates = []string{"brew"}
	case goos == "windows":
		candidates = []string{"winget", "scoop", "choco"}
	default:
		candidates = []string{"apt-get", "dnf", "pacman", "zypper", "apk", "brew", "pkg"}
	}

	found := make([]string, 0, len(candidates))
	for _, bin := range candidates {
		canonical := bin
		if bin == "apt-get" {
			canonical = "apt"
		}
		if _, err := exec.LookPath(bin); err == nil {
			found = append(found, canonical)
		} else if bin == "apk" {
			if _, sErr := os.Stat("/sbin/apk"); sErr == nil {
				found = append(found, "apk")
			} else if _, sErr := os.Stat("/usr/sbin/apk"); sErr == nil {
				found = append(found, "apk")
			}
		}
	}
	return found
}

func detectLinuxDistro() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		f, err = os.Open("/usr/lib/os-release")
		if err != nil {
			return "linux"
		}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id := strings.TrimPrefix(line, "ID=")
			id = strings.Trim(id, `"'`)
			return strings.ToLower(id)
		}
	}
	return "linux"
}

func detectWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	content := strings.ToLower(string(data))
	return strings.Contains(content, "microsoft") || strings.Contains(content, "wsl")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
