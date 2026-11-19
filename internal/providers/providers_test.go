package providers_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/manifest"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/providers"
	"github.com/arturonavax/env/internal/security"
)

func TestMultiStrategyDispatcherGitHubTarballFallbackToScript(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")

	osName := "linux"
	binExt := ""
	scriptInterp := "sh"
	if runtime.GOOS == "windows" {
		osName = "windows"
		binExt = ".cmd"
		scriptInterp = "cmd"
	}

	plat := platform.NewTestInfo(platform.TestInfoParams{
		OS:              osName,
		Arch:            "amd64",
		Distribution:    "ubuntu",
		PackageManagers: []string{},
		IsRoot:          true,
		HomeDir:         home,
		LocalBinDir:     binDir,
	})

	// Build an in-memory .tar.gz containing an executable binary "demo-tool"
	var tarBuf bytes.Buffer
	gzw := gzip.NewWriter(&tarBuf)
	tw := tar.NewWriter(gzw)
	binContent := []byte("#!/bin/sh\necho 'demo-tool 2.4.0'\n")
	if runtime.GOOS == "windows" {
		binContent = []byte("@echo off\r\necho demo-tool 2.4.0\r\n")
	}
	if err := tw.WriteHeader(&tar.Header{
		Name: "demo-2.4.0/demo-tool" + binExt,
		Mode: 0o755,
		Size: int64(len(binContent)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(binContent); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gzw.Close()

	sum := sha256.Sum256(tarBuf.Bytes())
	tarSHA := hex.EncodeToString(sum[:])

	scriptBody := "#!/bin/sh\nmkdir -p \"" + binDir + "\"\nprintf '#!/bin/sh\\necho script-tool 1.0.0\\n' > \"" + filepath.Join(binDir, "script-tool") + "\"\nchmod +x \"" + filepath.Join(binDir, "script-tool") + "\"\n"
	if runtime.GOOS == "windows" {
		escapedBinDir := filepath.Clean(binDir)
		targetFile := filepath.Join(escapedBinDir, "script-tool.cmd")
		scriptBody = "@echo off\r\nif not exist \"" + escapedBinDir + "\" mkdir \"" + escapedBinDir + "\"\r\necho @echo off> \"" + targetFile + "\"\r\necho echo script-tool 1.0.0>> \"" + targetFile + "\"\r\n"
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/demo-tarball.tar.gz":
			_, _ = w.Write(tarBuf.Bytes())
		case "/install-script":
			_, _ = w.Write([]byte(scriptBody))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dropIn := t.TempDir()
	yamlSpec := `packages:
  - id: demo-tool
    name: Demo Tool
    category: test
    binary_name: demo-tool` + binExt + `
    strategies:
      - type: github_release
        github_release:
          direct_urls:
            linux/amd64: "` + srv.URL + `/demo-tarball.tar.gz"
            windows/amd64: "` + srv.URL + `/demo-tarball.tar.gz"
          sha256: "` + tarSHA + `"
          binary_in_archive: "demo-tool` + binExt + `"
          install_to: "~/.local/bin/demo-tool` + binExt + `"
  - id: fallback-tool
    name: Fallback Tool
    category: test
    binary_name: script-tool` + binExt + `
    strategies:
      - type: github_release
        github_release:
          direct_urls:
            linux/amd64: "` + srv.URL + `/missing-404.tar.gz"
            windows/amd64: "` + srv.URL + `/missing-404.tar.gz"
          binary_in_archive: "script-tool` + binExt + `"
          install_to: "~/.local/bin/script-tool` + binExt + `"
      - type: script
        script:
          url: "` + srv.URL + `/install-script"
          interpreter: ` + scriptInterp + `
          creates: "~/.local/bin/script-tool` + binExt + `"
`
	if err := os.WriteFile(filepath.Join(dropIn, "test.yaml"), []byte(yamlSpec), 0o600); err != nil {
		t.Fatal(err)
	}

	cat, err := manifest.LoadCatalog(manifest.WithUserDropInDir(dropIn))
	if err != nil {
		t.Fatal(err)
	}

	runner := security.NewRunner(10 * time.Second)
	disp := providers.NewDispatcher(plat, cat, runner, "")

	// 1. Install demo-tool via verified tar.gz extraction
	demoRes := domain.ManagedResource{ID: "demo-tool", Name: "Demo Tool", BinaryName: "demo-tool" + binExt}
	if err := disp.Install(context.Background(), demoRes); err != nil {
		t.Fatalf("expected demo-tool install to succeed, got %v", err)
	}
	installed, ver, err := disp.IsInstalled(context.Background(), demoRes)
	if err != nil || !installed || ver != "2.4.0" {
		t.Fatalf("expected demo-tool installed at 2.4.0, got installed=%v ver=%q err=%v", installed, ver, err)
	}

	// 2. Install fallback-tool: tier 1 (404 tarball) fails -> falls back automatically to tier 2 (verified curl script)!
	fbRes := domain.ManagedResource{ID: "fallback-tool", Name: "Fallback Tool", BinaryName: "script-tool" + binExt}
	if err := disp.Install(context.Background(), fbRes); err != nil {
		t.Fatalf("expected fallback-tool multi-tier install to succeed, got %v", err)
	}
	installedFB, verFB, err := disp.IsInstalled(context.Background(), fbRes)
	if err != nil || !installedFB || verFB != "1.0.0" {
		t.Fatalf("expected fallback-tool installed at 1.0.0, got installed=%v ver=%q err=%v", installedFB, verFB, err)
	}

	// 3. Uninstall removes local binary cleanly
	if err := disp.Uninstall(context.Background(), demoRes); err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}
	installedAfter, _, _ := disp.IsInstalled(context.Background(), demoRes)
	if installedAfter {
		t.Fatal("expected demo-tool uninstalled")
	}
}

func TestDispatcherRejectsTarSlipAndUnsafeScriptGuardrails(t *testing.T) {
	home := t.TempDir()
	plat := platform.NewTestInfo(platform.TestInfoParams{
		OS:          "linux",
		Arch:        "amd64",
		HomeDir:     home,
		LocalBinDir: filepath.Join(home, ".local", "bin"),
	})

	// Build a malicious .tar.gz attempting path traversal outside extract root
	var badTar bytes.Buffer
	gzw := gzip.NewWriter(&badTar)
	tw := tar.NewWriter(gzw)
	payload := []byte("owned")
	_ = tw.WriteHeader(&tar.Header{
		Name: "../../escaped.txt",
		Mode: 0o644,
		Size: int64(len(payload)),
	})
	_, _ = tw.Write(payload)
	_ = tw.Close()
	_ = gzw.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/evil.tar.gz":
			_, _ = w.Write(badTar.Bytes())
		case "/evil-script.sh":
			_, _ = w.Write([]byte("#!/bin/sh\nrm -rf / --no-preserve-root\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dropIn := t.TempDir()
	yamlSpec := `packages:
  - id: evil-tar
    name: Evil Tar
    category: test
    strategies:
      - type: github_release
        github_release:
          direct_urls:
            linux/amd64: "` + srv.URL + `/evil.tar.gz"
          binary_in_archive: "evil"
  - id: evil-script
    name: Evil Script
    category: test
    strategies:
      - type: script
        script:
          url: "` + srv.URL + `/evil-script.sh"
          interpreter: sh
`
	if err := os.WriteFile(filepath.Join(dropIn, "evil.yaml"), []byte(yamlSpec), 0o600); err != nil {
		t.Fatal(err)
	}
	cat, err := manifest.LoadCatalog(manifest.WithUserDropInDir(dropIn))
	if err != nil {
		t.Fatal(err)
	}

	disp := providers.NewDispatcher(plat, cat, security.NewRunner(5*time.Second), "")

	if err := disp.Install(context.Background(), domain.ManagedResource{ID: "evil-tar", Name: "Evil Tar"}); err == nil {
		t.Fatal("expected Tar-Slip archive to be rejected")
	}
	if err := disp.Install(context.Background(), domain.ManagedResource{ID: "evil-script", Name: "Evil Script"}); err == nil {
		t.Fatal("expected destructive remote script to be blocked by guardrails")
	}
}

func TestDispatcherPreviewMultiDistroMatrix(t *testing.T) {
	cat, err := manifest.LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		os   string
		pm   string
		want string
	}{
		{name: "Debian apt fd-find", os: "linux", pm: "apt", want: "native:apt (fd-find)"},
		{name: "Arch pacman fd", os: "linux", pm: "pacman", want: "native:pacman (fd)"},
		{name: "Windows winget sharkdp.fd", os: "windows", pm: "winget", want: "native:winget (sharkdp.fd)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plat := platform.NewTestInfo(platform.TestInfoParams{
				OS:              tc.os,
				Arch:            "amd64",
				PackageManagers: []string{tc.pm},
				HomeDir:         t.TempDir(),
			})
			disp := providers.NewDispatcher(plat, cat, nil, "")
			if got := disp.PreviewStrategy("fd"); got != tc.want {
				t.Fatalf("PreviewStrategy(fd) = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestScriptStrategyCompletesWhenTargetCreatedEvenIfScriptHangs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping bash-specific hang test on Windows")
	}

	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	_ = os.MkdirAll(binDir, 0o755)

	targetFile := filepath.Join(binDir, "hanging-tool")

	// Script creates targetFile, then sleeps
	scriptBody := "#!/bin/sh\n" +
		"echo 'hanging-tool 1.0.0' > \"" + targetFile + "\"\n" +
		"chmod +x \"" + targetFile + "\"\n" +
		"sleep 30\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(scriptBody))
	}))
	defer srv.Close()

	dropIn := t.TempDir()
	yamlSpec := `packages:
  - id: hanging-tool
    name: Hanging Tool
    category: test
    binary_name: hanging-tool
    strategies:
      - type: script
        script:
          url: "` + srv.URL + `/install.sh"
          interpreter: sh
          creates: "` + targetFile + `"
`
	if err := os.WriteFile(filepath.Join(dropIn, "hanging.yaml"), []byte(yamlSpec), 0o600); err != nil {
		t.Fatal(err)
	}

	cat, err := manifest.LoadCatalog(manifest.WithUserDropInDir(dropIn))
	if err != nil {
		t.Fatal(err)
	}

	plat := platform.NewTestInfo(platform.TestInfoParams{
		OS:              "linux",
		Arch:            "amd64",
		PackageManagers: []string{},
		HomeDir:         home,
		LocalBinDir:     binDir,
	})

	disp := providers.NewDispatcher(plat, cat, security.NewRunner(30*time.Second), "")

	start := time.Now()
	err = disp.Install(context.Background(), domain.ManagedResource{ID: "hanging-tool", Name: "Hanging Tool"})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected install to succeed via creates watchdog, got: %v", err)
	}
	if elapsed > 10*time.Second {
		t.Fatalf("install took too long (%v), expected early completion after creates verified", elapsed)
	}
	if _, err := os.Stat(targetFile); err != nil {
		t.Fatalf("expected target file %s to exist, stat error: %v", targetFile, err)
	}
}

func TestDispatcherEffectiveDependenciesByHost(t *testing.T) {
	cat, err := manifest.LoadCatalog()
	if err != nil {
		t.Fatalf("LoadCatalog failed: %v", err)
	}

	tests := []struct {
		name      string
		params    platform.TestInfoParams
		isolate   bool
		createBin string
		pkgID     string
		want      []string
	}{
		{
			name: "pyenv on linux apt uses linux build deps",
			params: platform.TestInfoParams{
				OS:              "linux",
				Arch:            "amd64",
				PackageManagers: []string{"apt"},
			},
			pkgID: "pyenv",
			want:  []string{"git", "curl", "build-essential", "bz2-dev", "ssl-dev", "zlib-dev", "readline-dev", "sqlite-dev", "ffi-dev", "xz-dev", "ncurses-dev"},
		},
		{
			name: "pyenv on darwin brew has no source build deps",
			params: platform.TestInfoParams{
				OS:              "darwin",
				Arch:            "arm64",
				PackageManagers: []string{"brew"},
			},
			pkgID: "pyenv",
			want:  []string{},
		},
		{
			name: "rust on linux uses curl and build-essential",
			params: platform.TestInfoParams{
				OS:              "linux",
				Arch:            "amd64",
				PackageManagers: []string{"apt"},
			},
			pkgID: "rust",
			want:  []string{"curl", "build-essential"},
		},
		{
			name: "rust on darwin uses curl without linux build-essential",
			params: platform.TestInfoParams{
				OS:              "darwin",
				Arch:            "arm64",
				PackageManagers: []string{"brew"},
			},
			pkgID: "rust",
			want:  []string{"curl"},
		},
		{
			name: "rust on windows winget filters out linux-only build-essential",
			params: platform.TestInfoParams{
				OS:              "windows",
				Arch:            "amd64",
				PackageManagers: []string{"winget"},
			},
			pkgID: "rust",
			want:  []string{},
		},
		{
			name: "claude-code on linux uses script installer with curl only",
			params: platform.TestInfoParams{
				OS:              "linux",
				Arch:            "amd64",
				PackageManagers: []string{"apt"},
			},
			pkgID: "claude-code",
			want:  []string{"curl"},
		},
		{
			name: "claude-code on darwin brew uses cask with zero deps",
			params: platform.TestInfoParams{
				OS:              "darwin",
				Arch:            "arm64",
				PackageManagers: []string{"brew"},
			},
			pkgID: "claude-code",
			want:  []string{},
		},
		{
			name: "codex-cli on linux uses github release with zero pnpm or curl deps",
			params: platform.TestInfoParams{
				OS:              "linux",
				Arch:            "amd64",
				PackageManagers: []string{"apt"},
			},
			pkgID: "codex-cli",
			want:  []string{},
		},
		{
			name: "tree-sitter-cli on alpine uses native apk with zero rust dep",
			params: platform.TestInfoParams{
				OS:              "linux",
				Arch:            "amd64",
				Distribution:    "alpine",
				PackageManagers: []string{"apk"},
			},
			pkgID: "tree-sitter-cli",
			want:  []string{},
		},
		{
			name: "tree-sitter-cli on opensuse without pnpm falls back to cargo with rust dep",
			params: platform.TestInfoParams{
				OS:              "linux",
				Arch:            "amd64",
				Distribution:    "opensuse-tumbleweed",
				PackageManagers: []string{"zypper"},
			},
			isolate: true,
			pkgID:   "tree-sitter-cli",
			want:    []string{"rust"},
		},
		{
			name: "tree-sitter-cli on opensuse with pnpm uses pnpm tier instead of rust",
			params: platform.TestInfoParams{
				OS:              "linux",
				Arch:            "amd64",
				Distribution:    "opensuse-tumbleweed",
				PackageManagers: []string{"zypper"},
			},
			isolate:   true,
			createBin: "pnpm",
			pkgID:     "tree-sitter-cli",
			want:      []string{"pnpm"},
		},
		{
			name: "tree-sitter-cli on termux without pnpm falls back to cargo with rust dep",
			params: platform.TestInfoParams{
				OS:              "android",
				Arch:            "arm64",
				PackageManagers: []string{"pkg"},
			},
			isolate: true,
			pkgID:   "tree-sitter-cli",
			want:    []string{"rust"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.isolate {
				t.Setenv("PATH", t.TempDir())
			}
			home := t.TempDir()
			binDir := filepath.Join(home, ".local", "bin")
			if tc.createBin != "" {
				if err := os.MkdirAll(binDir, 0o755); err != nil {
					t.Fatal(err)
				}
				binName := tc.createBin
				if tc.params.OS == "windows" {
					binName += ".exe"
				}
				if err := os.WriteFile(filepath.Join(binDir, binName), []byte("#!/bin/sh\n"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			params := tc.params
			params.HomeDir = home
			params.LocalBinDir = binDir
			plat := platform.NewTestInfo(params)
			disp := providers.NewDispatcher(plat, cat, security.NewRunner(5*time.Second), "")

			got := disp.EffectiveDependencies(tc.pkgID)
			if !slices.Equal(got, tc.want) {
				t.Errorf("EffectiveDependencies(%q) = %v, want %v", tc.pkgID, got, tc.want)
			}
		})
	}
}

func TestDispatcherPreviewStrategyDisplayServer(t *testing.T) {
	cat, err := manifest.LoadCatalog()
	if err != nil {
		t.Fatalf("LoadCatalog failed: %v", err)
	}

	homeWayland := t.TempDir()
	platWayland := platform.NewTestInfo(platform.TestInfoParams{
		OS:              "linux",
		Arch:            "amd64",
		PackageManagers: []string{"apt"},
		DisplayServer:   "wayland",
		HomeDir:         homeWayland,
	})
	dispWayland := providers.NewDispatcher(platWayland, cat, security.NewRunner(5*time.Second), "")
	if got := dispWayland.PreviewStrategy("xclip"); got != "unsupported" {
		t.Errorf("PreviewStrategy(xclip) on wayland = %q, want %q", got, "unsupported")
	}

	homeX11 := t.TempDir()
	platX11 := platform.NewTestInfo(platform.TestInfoParams{
		OS:              "linux",
		Arch:            "amd64",
		PackageManagers: []string{"apt"},
		DisplayServer:   "x11",
		HomeDir:         homeX11,
	})
	dispX11 := providers.NewDispatcher(platX11, cat, security.NewRunner(5*time.Second), "")
	if got := dispX11.PreviewStrategy("xclip"); got != "native:apt (xclip)" {
		t.Errorf("PreviewStrategy(xclip) on x11 = %q, want %q", got, "native:apt (xclip)")
	}
}

func TestDispatcherNativeBrewCaskInstalledCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell stub for brew requires non-Windows host")
	}

	home := t.TempDir()
	fakeBin := t.TempDir()
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	brewScript := `#!/bin/sh
if [ "$1" = "list" ] && [ "$2" = "--cask" ] && [ "$3" = "--versions" ] && [ "$4" = "amethyst" ]; then
  echo "amethyst 0.24.0"
  exit 0
fi
exit 1
`
	if err := os.WriteFile(filepath.Join(fakeBin, "brew"), []byte(brewScript), 0o755); err != nil {
		t.Fatal(err)
	}

	cat, err := manifest.LoadCatalog()
	if err != nil {
		t.Fatalf("LoadCatalog failed: %v", err)
	}

	plat := platform.NewTestInfo(platform.TestInfoParams{
		OS:              "darwin",
		Arch:            "arm64",
		PackageManagers: []string{"brew"},
		HomeDir:         home,
		LocalBinDir:     filepath.Join(home, ".local", "bin"),
	})
	disp := providers.NewDispatcher(plat, cat, security.NewRunner(5*time.Second), "")

	installed, ver, err := disp.IsInstalled(context.Background(), domain.ManagedResource{ID: "amethyst", Name: "Amethyst"})
	if err != nil {
		t.Fatalf("IsInstalled(amethyst) error: %v", err)
	}
	if !installed {
		t.Fatalf("expected brew cask amethyst to be detected as installed, got installed=false ver=%q", ver)
	}

	notInstalled, _, err := disp.IsInstalled(context.Background(), domain.ManagedResource{ID: "karabiner-elements", Name: "Karabiner-Elements"})
	if err != nil {
		t.Fatalf("IsInstalled(karabiner-elements) error: %v", err)
	}
	if notInstalled {
		t.Errorf("expected missing brew cask karabiner-elements not to report installed")
	}
}

func TestDispatcherFallbackInstallsStrategyDependsOn(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")

	osName := "linux"
	binExt := ""
	scriptInterp := "sh"
	if runtime.GOOS == "windows" {
		osName = "windows"
		binExt = ".cmd"
		scriptInterp = "cmd"
	}

	helperPath := filepath.Join(binDir, "helper-dep"+binExt)
	unusedPath := filepath.Join(binDir, "unused-dep"+binExt)
	targetPath := filepath.Join(binDir, "tier-target"+binExt)

	mkScript := func(path, line string) string {
		if runtime.GOOS == "windows" {
			return "@echo off\r\nif not exist \"" + binDir + "\" mkdir \"" + binDir + "\"\r\necho @echo off> \"" + path + "\"\r\necho " + line + ">> \"" + path + "\"\r\n"
		}
		return "#!/bin/sh\nmkdir -p \"" + binDir + "\"\nprintf '#!/bin/sh\\n" + line + "\\n' > \"" + path + "\"\nchmod +x \"" + path + "\"\n"
	}

	targetScript := mkScript(targetPath, "echo tier-target 1.0.0")
	if runtime.GOOS != "windows" {
		targetScript = "#!/bin/sh\nif [ ! -f \"" + helperPath + "\" ]; then echo 'missing helper-dep' >&2; exit 1; fi\n" + mkScript(targetPath, "echo tier-target 1.0.0")
	} else {
		targetScript = "@echo off\r\nif not exist \"" + helperPath + "\" exit /b 1\r\n" + mkScript(targetPath, "echo tier-target 1.0.0")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/install-helper":
			_, _ = w.Write([]byte(mkScript(helperPath, "echo helper-dep 1.0.0")))
		case "/install-unused":
			_, _ = w.Write([]byte(mkScript(unusedPath, "echo unused-dep 1.0.0")))
		case "/install-target":
			_, _ = w.Write([]byte(targetScript))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	plat := platform.NewTestInfo(platform.TestInfoParams{
		OS:              osName,
		Arch:            "amd64",
		PackageManagers: []string{},
		HomeDir:         home,
		LocalBinDir:     binDir,
	})

	dropIn := t.TempDir()
	yamlSpec := `packages:
  - id: helper-dep
    name: Helper Dep
    category: test
    binary_name: helper-dep` + binExt + `
    strategies:
      - type: script
        script:
          url: "` + srv.URL + `/install-helper"
          interpreter: ` + scriptInterp + `
          creates: "~/.local/bin/helper-dep` + binExt + `"
  - id: unused-dep
    name: Unused Dep
    category: test
    binary_name: unused-dep` + binExt + `
    strategies:
      - type: script
        script:
          url: "` + srv.URL + `/install-unused"
          interpreter: ` + scriptInterp + `
          creates: "~/.local/bin/unused-dep` + binExt + `"
  - id: tier-target
    name: Tier Target
    category: test
    binary_name: tier-target` + binExt + `
    strategies:
      - type: github_release
        github_release:
          direct_urls:
            linux/amd64: "` + srv.URL + `/missing-404.tar.gz"
            windows/amd64: "` + srv.URL + `/missing-404.tar.gz"
          binary_in_archive: "tier-target` + binExt + `"
          install_to: "~/.local/bin/tier-target` + binExt + `"
      - type: native
        depends_on: [unused-dep]
        packages:
          pacman: [tier-target]
      - type: script
        depends_on: [helper-dep]
        script:
          url: "` + srv.URL + `/install-target"
          interpreter: ` + scriptInterp + `
          creates: "~/.local/bin/tier-target` + binExt + `"
`
	if err := os.WriteFile(filepath.Join(dropIn, "fallback_deps.yaml"), []byte(yamlSpec), 0o600); err != nil {
		t.Fatal(err)
	}

	cat, err := manifest.LoadCatalog(manifest.WithUserDropInDir(dropIn))
	if err != nil {
		t.Fatalf("LoadCatalog failed: %v", err)
	}

	disp := providers.NewDispatcher(plat, cat, security.NewRunner(10*time.Second), "")
	targetRes := domain.ManagedResource{ID: "tier-target", Name: "Tier Target", BinaryName: "tier-target" + binExt}
	if err := disp.Install(context.Background(), targetRes); err != nil {
		t.Fatalf("expected fallback install to install helper-dep and succeed, got: %v", err)
	}

	if _, err := os.Stat(helperPath); err != nil {
		t.Errorf("expected fallback strategy dependency helper-dep to be installed at %s: %v", helperPath, err)
	}
	if _, err := os.Stat(unusedPath); !os.IsNotExist(err) {
		t.Errorf("ineligible native tier dependency unused-dep should not have been installed, stat err=%v", err)
	}
	if _, err := os.Stat(targetPath); err != nil {
		t.Errorf("expected tier-target to be installed at %s: %v", targetPath, err)
	}
}
