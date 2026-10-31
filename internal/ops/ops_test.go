package ops_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/arturonavax/env/internal/ops"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
)

var mockToolBin string

func TestMain(m *testing.M) {
	tempDir, err := os.MkdirTemp("", "mock-tool-build-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	exeName := "mock_tool"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	mockToolBin = filepath.Join(tempDir, exeName)

	cmd := exec.Command("go", "build", "-o", mockToolBin, "./testdata/mock_tool")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build mock tool: %v, output: %s\n", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func installMockTool(t *testing.T, dir, name string) {
	t.Helper()
	data, err := os.ReadFile(mockToolBin)
	if err != nil {
		t.Fatalf("reading mock tool binary: %v", err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o755); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		pExe := filepath.Join(dir, name+".exe")
		if err := os.WriteFile(pExe, data, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRebuilderDryRunAndMockExecution(t *testing.T) {
	homeDir := t.TempDir()
	mockBinDir := t.TempDir()
	goBinDir := filepath.Join(homeDir, "go", "bin")
	logFile := filepath.Join(t.TempDir(), "invocations.log")

	if err := os.MkdirAll(goBinDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Seed fake compiled Go binaries in ~/go/bin.
	for _, binName := range []string{"gopls", "gofumpt", "localdev"} {
		p := filepath.Join(goBinDir, binName)
		if err := os.WriteFile(p, []byte("ELF-mock"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Seed zsh startup files in homeDir.
	for _, zf := range []string{".zshrc", ".base.zsh"} {
		if err := os.WriteFile(filepath.Join(homeDir, zf), []byte("# zsh config\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	installMockTool(t, mockBinDir, "go")
	installMockTool(t, mockBinDir, "cargo")
	installMockTool(t, mockBinDir, "rustup")
	installMockTool(t, mockBinDir, "pnpm")
	installMockTool(t, mockBinDir, "pipx")
	installMockTool(t, mockBinDir, "nvim")
	installMockTool(t, mockBinDir, "zsh")

	plat := platform.NewTestInfo(platform.TestInfoParams{
		OS:          "linux",
		Arch:        "amd64",
		HomeDir:     homeDir,
		LocalBinDir: filepath.Join(homeDir, ".local", "bin"),
	})
	runner := security.NewRunner(10 * time.Second)
	env := []string{
		"PATH=" + mockBinDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + homeDir,
		"GOBIN=" + goBinDir,
		"ANX_MOCK_LOG_FILE=" + logFile,
	}

	t.Run("dry-run previews commands without mutating or invoking installers", func(t *testing.T) {
		rebuilder := ops.NewRebuilder(plat, runner, ops.WithDryRun(true), ops.WithEnv(env))
		results, err := rebuilder.Rebuild(context.Background(), []string{"all"})
		if err != nil {
			t.Fatalf("Rebuild dry-run failed: %v", err)
		}
		if len(results) != 6 {
			t.Fatalf("len(results) = %d, want 6", len(results))
		}
		if _, err := os.Stat(logFile); !os.IsNotExist(err) {
			raw, _ := os.ReadFile(logFile)
			t.Fatalf("expected zero mutating commands logged during dry-run, got:\n%s", string(raw))
		}

		byScope := map[string]ops.RebuildResult{}
		for _, res := range results {
			byScope[res.Scope] = res
		}
		if byScope["go"].Rebuilt != 2 {
			t.Errorf("go dry-run discovered count = %d, want 2 (cmds: %v)", byScope["go"].Rebuilt, byScope["go"].Commands)
		}
		if byScope["rust"].Rebuilt != 2 {
			t.Errorf("rust dry-run discovered count = %d, want 2 (cmds: %v)", byScope["rust"].Rebuilt, byScope["rust"].Commands)
		}
	})

	t.Run("live execution runs bounded go installs and all ecosystem commands", func(t *testing.T) {
		rebuilder := ops.NewRebuilder(plat, runner, ops.WithEnv(env))
		results, err := rebuilder.Rebuild(context.Background(), []string{"all"})
		if err != nil {
			t.Fatalf("Rebuild execution failed: %v", err)
		}
		if len(results) != 6 {
			t.Fatalf("len(results) = %d, want 6", len(results))
		}

		rawLog, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("reading invocation log: %v", err)
		}
		logged := string(rawLog)

		for _, expected := range []string{
			"golang.org/x/tools/gopls@latest",
			"mvdan.cc/gofumpt@latest",
			"cargo install difftastic",
			"cargo install git-delta",
			"pnpm update -g",
			"pipx reinstall-all",
			"nvim --headless +Lazy! sync +qa",
			"nvim --headless +MasonUpdate +qa",
			"zcompile",
		} {
			if !strings.Contains(logged, expected) {
				t.Errorf("invocation log missing %q; full log:\n%s", expected, logged)
			}
		}

		if strings.Contains(logged, "command-line-arguments") || strings.Contains(logged, "local-tool") {
			t.Errorf("local non-registry packages should have been skipped; log:\n%s", logged)
		}
	})
}

func TestOptimizerAuditSynthesizeAndGuardrails(t *testing.T) {
	homeDir := t.TempDir()
	procRoot := t.TempDir()
	sysRoot := t.TempDir()
	mockBinDir := t.TempDir()

	// Seed synthetic /proc and /sys files for deterministic audit verification.
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

	mustWrite(procRoot, "cpuinfo", "processor\t: 0\nmodel name\t: Intel(R) Core(TM) i7-4910MQ CPU @ 2.90GHz\n\nprocessor\t: 1\nmodel name\t: Intel(R) Core(TM) i7-4910MQ CPU @ 2.90GHz\n")
	mustWrite(procRoot, "meminfo", "MemTotal:       32768000 kB\nMemAvailable:   24576000 kB\nSwapTotal:       8192000 kB\nSwapFree:        8192000 kB\n")
	mustWrite(procRoot, "sys/vm/swappiness", "60\n")
	mustWrite(procRoot, "sys/fs/inotify/max_user_watches", "8192\n")
	mustWrite(procRoot, "sys/net/ipv4/tcp_congestion_control", "cubic\n")
	mustWrite(procRoot, "sys/kernel/osrelease", "6.12.0-generic\n")
	mustWrite(procRoot, "mounts", "/dev/nvme0n1p2 / ext4 rw,relatime 0 0\ntmpfs /run tmpfs rw 0 0\n")

	mustWrite(sysRoot, "block/nvme0n1/queue/scheduler", "[none] mq-deadline kyber\n")
	mustWrite(sysRoot, "block/nvme0n1/queue/rotational", "0\n")
	mustWrite(sysRoot, "devices/system/cpu/cpu0/cpufreq/scaling_governor", "powersave\n")

	installMockTool(t, mockBinDir, "lynis")
	installMockTool(t, mockBinDir, "bash")

	plat := platform.NewTestInfo(platform.TestInfoParams{
		OS:           "linux",
		Arch:         "amd64",
		Distribution: "ubuntu",
		HomeDir:      homeDir,
	})
	runner := security.NewRunner(10 * time.Second)
	env := []string{
		"PATH=" + mockBinDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + homeDir,
		"XDG_SESSION_TYPE=x11",
		"XDG_CURRENT_DESKTOP=XFCE",
	}

	opt := ops.NewOptimizer(
		plat,
		runner,
		ops.WithOptimizerEnv(env),
		ops.WithFSRoot(procRoot, sysRoot),
	)

	report, err := opt.AuditSystem(context.Background(), false)
	if err != nil {
		t.Fatalf("AuditSystem failed: %v", err)
	}
	if !strings.Contains(report.CPUModel, "i7-4910MQ") || report.CPUCores != 2 {
		t.Errorf("unexpected CPU telemetry: model=%q cores=%d", report.CPUModel, report.CPUCores)
	}
	if report.MemoryTotalMB != 32000 || report.MemoryFreeMB != 24000 {
		t.Errorf("unexpected memory telemetry: total=%d free=%d", report.MemoryTotalMB, report.MemoryFreeMB)
	}
	if report.Swappiness != 60 || report.InotifyMaxWatches != 8192 {
		t.Errorf("unexpected sysctl telemetry: swappiness=%d inotify=%d", report.Swappiness, report.InotifyMaxWatches)
	}
	if len(report.DiskMounts) != 1 || report.DiskMounts[0].Scheduler != "none" {
		t.Errorf("unexpected disk mounts: %+v", report.DiskMounts)
	}
	if !report.LynisRan || len(report.LynisWarnings) == 0 || len(report.LynisSuggestions) == 0 {
		t.Errorf("expected Lynis warnings & suggestions distilled, got ran=%v warn=%v sug=%v",
			report.LynisRan, report.LynisWarnings, report.LynisSuggestions)
	}
	if len(report.Findings) == 0 {
		t.Errorf("expected non-empty heuristic findings for swappiness=60 and inotify=8192")
	}

	t.Run("offline baseline script generator produces valid guardrail-clean script", func(t *testing.T) {
		script, err := opt.SynthesizeScript(context.Background(), report)
		if err != nil {
			t.Fatalf("SynthesizeScript baseline failed: %v", err)
		}
		if !strings.HasPrefix(script, "#!/usr/bin/env bash") {
			t.Errorf("expected bash shebang, got:\n%s", script)
		}
		if violations := opt.ValidateScript(script); len(violations) > 0 {
			t.Fatalf("baseline script triggered guardrails: %+v", violations)
		}
	})

	t.Run("custom AI PromptFunc synthesizes clean script and executes safely", func(t *testing.T) {
		markerFile := filepath.Join(t.TempDir(), "applied.txt")
		aiOpt := ops.NewOptimizer(
			plat,
			runner,
			ops.WithOptimizerEnv(env),
			ops.WithFSRoot(procRoot, sysRoot),
			ops.WithPromptFunc(func(ctx context.Context, prompt string) (string, error) {
				if !strings.Contains(prompt, "i7-4910MQ") {
					t.Errorf("expected hardware audit details in prompt, got:\n%s", prompt)
				}
				return "```bash\n#!/usr/bin/env bash\nset -euo pipefail\necho 'optimized' > " + markerFile + "\n```", nil
			}),
		)

		script, err := aiOpt.SynthesizeScript(context.Background(), report)
		if err != nil {
			t.Fatalf("AI SynthesizeScript failed: %v", err)
		}
		if _, err := aiOpt.ExecuteScript(context.Background(), script, false); err != nil {
			t.Fatalf("ExecuteScript failed: %v", err)
		}
		got, err := os.ReadFile(markerFile)
		if err != nil || strings.TrimSpace(string(got)) != "optimized" {
			t.Fatalf("expected script execution side effect, got %q (err=%v)", string(got), err)
		}
	})

	t.Run("guardrail engine blocks destructive AI or user scripts before execution", func(t *testing.T) {
		unsafeOpt := ops.NewOptimizer(
			plat,
			runner,
			ops.WithPromptFunc(func(ctx context.Context, prompt string) (string, error) {
				return "#!/usr/bin/env bash\nexit 0\nrm -rf /etc\n", nil
			}),
		)

		_, err := unsafeOpt.SynthesizeScript(context.Background(), report)
		if !errors.Is(err, ops.ErrGuardrailViolation) {
			t.Fatalf("expected ErrGuardrailViolation from SynthesizeScript, got: %v", err)
		}

		_, execErr := unsafeOpt.ExecuteScript(context.Background(), "#!/usr/bin/env bash\nexit 0\ndd if=/dev/zero of=/dev/nvme0n1\n", false)
		if !errors.Is(execErr, ops.ErrGuardrailViolation) {
			t.Fatalf("expected ErrGuardrailViolation from ExecuteScript, got: %v", execErr)
		}
	})
}
