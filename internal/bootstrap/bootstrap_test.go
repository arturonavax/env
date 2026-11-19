package bootstrap_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arturonavax/env/internal/bootstrap"
)

func TestNewContainer(t *testing.T) {
	c, err := bootstrap.NewContainer(
		bootstrap.WithVersion("1.0.0-test", "abcdef", "2026-10-30"),
		bootstrap.WithOffline(true),
	)
	if err != nil {
		t.Fatalf("failed to create bootstrap container: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	if c.Platform == nil {
		t.Errorf("expected platform info in container")
	}
	if c.Catalog == nil {
		t.Errorf("expected catalog in container")
	}
	if c.InstallSvc == nil {
		t.Errorf("expected InstallSvc in container")
	}
	if c.SyncSvc == nil {
		t.Errorf("expected SyncSvc in container")
	}
	if c.Version != "1.0.0-test" {
		t.Errorf("expected version '1.0.0-test', got %q", c.Version)
	}
}

func TestResolveRepoRoot(t *testing.T) {
	tmpDir := t.TempDir()
	manifestsDir := filepath.Join(tmpDir, "manifests")
	filesDir := filepath.Join(tmpDir, "files")
	if err := os.MkdirAll(manifestsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	resolved := bootstrap.ResolveRepoRoot(tmpDir, "", "")
	if resolved != tmpDir {
		t.Errorf("expected resolved repo root %q, got %q", tmpDir, resolved)
	}

	// When configured is empty and cwd is not a repo (or when testing explicit configured)
	otherDir := filepath.Join(tmpDir, "other")
	if err := os.MkdirAll(filepath.Join(otherDir, "manifests"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(otherDir, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	resolvedOther := bootstrap.ResolveRepoRoot(otherDir, "", "")
	if resolvedOther != otherDir {
		t.Errorf("expected resolved configured %q, got %q", otherDir, resolvedOther)
	}
}

func TestMacOSFreshBookBootstrapWithoutCLT(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell bootstrap test skipped on Windows")
	}

	repoRoot := filepath.Join("..", "..")

	t.Run("scripts avoid GNU sort -V or --version-sort unsupported on macOS BSD sort", func(t *testing.T) {
		for _, rel := range []string{"install.sh", "scripts/bootstrap_anx.sh", "scripts/install_golang.sh"} {
			raw, err := os.ReadFile(filepath.Join(repoRoot, rel))
			if err != nil {
				t.Fatal(err)
			}
			txt := string(raw)
			if strings.Contains(txt, "sort -V") || strings.Contains(txt, "--version-sort") {
				t.Errorf("%s uses GNU sort version flag which fails on stock macOS BSD sort", rel)
			}
		}
	})

	t.Run("install.sh detects Apple /usr/bin/git shim on Darwin without CLT and falls back cleanly to tarball twice", func(t *testing.T) {
		// Create minimal tarball payload env-main/scripts/bootstrap_anx.sh + manifests/
		var tarBuf bytes.Buffer
		gz := gzip.NewWriter(&tarBuf)
		tw := tar.NewWriter(gz)
		files := map[string]string{
			"env-main/scripts/bootstrap_anx.sh": "#!/usr/bin/env bash\necho \"BOOTSTRAP_OK $*\"\n",
			"env-main/manifests/configs.yaml":   "configurations: []\n",
		}
		for name, body := range files {
			if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body))}); err != nil {
				t.Fatal(err)
			}
			if _, err := tw.Write([]byte(body)); err != nil {
				t.Fatal(err)
			}
		}
		_ = tw.Close()
		_ = gz.Close()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(tarBuf.Bytes())
		}))
		defer srv.Close()

		home := t.TempDir()
		binDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		targetRepo := filepath.Join(home, ".local", "share", "anx", "repo")

		writeStub := func(name, body string) {
			if err := os.WriteFile(filepath.Join(binDir, name), []byte(body), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		writeStub("uname", "#!/bin/sh\n[ \"$1\" = \"-m\" ] && echo arm64 || echo Darwin\n")
		writeStub("xcode-select", "#!/bin/sh\nexit 2\n")
		writeStub("git", `#!/bin/sh
echo "SHIM_GIT_CALLED $*" >> "`+logFile+`"
echo "xcode-select: note: No developer tools were found, requesting install." >&2
exit 1
`)

		// Run install.sh from an empty cwd (simulating `curl ... | sh` remote bootstrap) twice.
		installScriptPath, err := filepath.Abs(filepath.Join(repoRoot, "install.sh"))
		if err != nil {
			t.Fatal(err)
		}
		for pass := 1; pass <= 2; pass++ {
			cmd := exec.Command("sh", installScriptPath, "--from-source", "--plan")
			cmd.Dir = home
			cmd.Env = []string{
				"HOME=" + home,
				"PATH=" + binDir + ":/usr/bin:/bin",
				"ANX_REPO_DIR=" + targetRepo,
				"ANX_TARBALL_URL=" + srv.URL + "/main.tar.gz",
				"ANX_SYSTEM_GIT_PATH=" + filepath.Join(binDir, "git"),
				"ANX_SKIP_CLT=1",
				"ANX_SKIP_HOMEBREW=1",
			}
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("pass %d: install.sh failed: %v\nout:\n%s", pass, err, out)
			}
			if !strings.Contains(string(out), "BOOTSTRAP_OK") {
				t.Errorf("pass %d: expected BOOTSTRAP_OK from extracted tarball script, got:\n%s", pass, out)
			}
		}
		if calls, err := os.ReadFile(logFile); err == nil && strings.Contains(string(calls), "SHIM_GIT_CALLED") {
			t.Errorf("install.sh must not invoke uninitialized Apple git shim, log:\n%s", calls)
		}
	})

	t.Run("bootstrap_anx.sh installs latest CLT headlessly on Darwin without sort -V and compiles pure-Go cleanly if CLT falls back", func(t *testing.T) {
		home := t.TempDir()
		binDir := t.TempDir()
		logFile := filepath.Join(t.TempDir(), "calls.log")
		cltFlag := filepath.Join(t.TempDir(), "clt.installed")
		placeholder := filepath.Join(t.TempDir(), "clt.placeholder")
		fakeDevDir := filepath.Join(t.TempDir(), "CommandLineTools")

		writeStub := func(name, body string) {
			if err := os.WriteFile(filepath.Join(binDir, name), []byte(body), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		writeStub("uname", "#!/bin/sh\n[ \"$1\" = \"-m\" ] && echo arm64 || echo Darwin\n")
		writeStub("xcode-select", `#!/bin/sh
if [ "$1" = "-p" ]; then
  if [ -f "`+cltFlag+`" ]; then
    echo "`+fakeDevDir+`"
    exit 0
  fi
  exit 2
fi
echo "xcode-select $*" >> "`+logFile+`"
exit 0
`)
		writeStub("softwareupdate", `#!/bin/sh
echo "softwareupdate $*" >> "`+logFile+`"
if [ "$1" = "-l" ]; then
  echo "Software Update found the following new or updated software:"
  echo "   * Label: Command Line Tools for Xcode-16.1"
  echo "	Title: Command Line Tools for Xcode, Version: 16.1, Size: 800000KiB, Recommended: YES,"
  echo "   * Label: Command Line Tools for Xcode-27.0"
  echo "	Title: Command Line Tools for Xcode, Version: 27.0, Size: 840000KiB, Recommended: YES,"
  exit 0
fi
if [ "$1" = "-i" ]; then
  mkdir -p "`+fakeDevDir+`/usr/bin"
  printf '#!/bin/sh\nexit 0\n' > "`+fakeDevDir+`/usr/bin/git"
  chmod +x "`+fakeDevDir+`/usr/bin/git"
  touch "`+cltFlag+`"
  exit 0
fi
exit 0
`)
		writeStub("sudo", `#!/bin/sh
if [ "$1" = "-v" ] || [ "$1" = "-n" ]; then exit 0; fi
echo "sudo $*" >> "`+logFile+`"
"$@"
`)
		writeStub("brew", "#!/bin/sh\nexit 0\n")
		writeStub("go", `#!/bin/sh
if [ "$1" = "version" ]; then
  echo "go version go1.27.3 darwin/arm64"
  exit 0
fi
if [ "$1" = "build" ]; then
  out=""
  prev=""
  for a in "$@"; do
    if [ "$prev" = "-o" ]; then out="$a"; fi
    prev="$a"
  done
  printf '#!/bin/sh\necho "ANX_STUB $*"\n' > "$out"
  chmod +x "$out"
  exit 0
fi
exit 1
`)
		writeStub("git", `#!/bin/sh
echo "SHIM_GIT $*" >> "`+logFile+`"
exit 1
`)

		cmd := exec.Command("bash", filepath.Join(repoRoot, "scripts", "bootstrap_anx.sh"), "--from-source", "--version")
		cmd.Env = []string{
			"HOME=" + home,
			"PATH=" + binDir + ":/usr/bin:/bin",
			"ANX_SYSTEM_GIT_PATH=" + filepath.Join(binDir, "git"),
			"ANX_CLT_PLACEHOLDER=" + placeholder,
			"ANX_SKIP_HOMEBREW=1",
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("bootstrap_anx.sh failed: %v\nout:\n%s", err, out)
		}
		callsBytes, _ := os.ReadFile(logFile)
		calls := string(callsBytes)
		if !strings.Contains(calls, "sudo softwareupdate -i Command Line Tools for Xcode-27.0") {
			t.Errorf("expected latest indented CLT label 27.0 to be installed headlessly, calls:\n%s\nout:\n%s", calls, out)
		}
		if strings.Contains(calls, "Command Line Tools for Xcode-16.1") {
			t.Errorf("should select newer CLT 27.0 over 16.1, calls:\n%s", calls)
		}
		if _, err := os.Stat(placeholder); !os.IsNotExist(err) {
			t.Errorf("expected CLT placeholder sentinel %s to be cleaned up", placeholder)
		}
		if !strings.Contains(string(out), "ANX_STUB --version") {
			t.Errorf("expected compiled ANX stub execution, got:\n%s", out)
		}

		// Also verify fallback when CLT stays absent skips shim git and uses CGO_ENABLED=0 cleanly.
		_ = os.Remove(cltFlag)
		_ = os.Remove(filepath.Join(fakeDevDir, "usr", "bin", "git"))
		_ = os.WriteFile(logFile, []byte(""), 0o644)
		writeStub("go", `#!/bin/sh
if [ "$1" = "version" ]; then
  echo "go version go1.27.3 darwin/arm64"
  exit 0
fi
if [ "$1" = "build" ]; then
  echo "GO_BUILD CGO_ENABLED=${CGO_ENABLED:-unset}" >> "`+logFile+`"
  out=""
  prev=""
  for a in "$@"; do
    if [ "$prev" = "-o" ]; then out="$a"; fi
    prev="$a"
  done
  printf '#!/bin/sh\necho "ANX_PURE_GO $*"\n' > "$out"
  chmod +x "$out"
  exit 0
fi
exit 1
`)
		cmdNoCLT := exec.Command("bash", filepath.Join(repoRoot, "scripts", "bootstrap_anx.sh"), "--from-source", "--version")
		cmdNoCLT.Env = []string{
			"HOME=" + home,
			"PATH=" + binDir + ":/usr/bin:/bin",
			"ANX_SYSTEM_GIT_PATH=" + filepath.Join(binDir, "git"),
			"ANX_SKIP_CLT=1",
			"ANX_SKIP_HOMEBREW=1",
		}
		outNoCLT, err := cmdNoCLT.CombinedOutput()
		if err != nil {
			t.Fatalf("bootstrap_anx.sh fallback without CLT failed: %v\nout:\n%s", err, outNoCLT)
		}
		calls2Bytes, _ := os.ReadFile(logFile)
		calls2 := string(calls2Bytes)
		if strings.Contains(calls2, "SHIM_GIT") {
			t.Errorf("bootstrap_anx.sh must not invoke git shim when CLT is absent, calls:\n%s", calls2)
		}
		if !strings.Contains(calls2, "GO_BUILD CGO_ENABLED=0") {
			t.Errorf("expected CGO_ENABLED=0 when CLT is missing on Darwin, calls:\n%s", calls2)
		}
	})
}
