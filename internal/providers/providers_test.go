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
