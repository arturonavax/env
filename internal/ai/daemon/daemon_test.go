package daemon_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/ai/daemon"
	"github.com/arturonavax/env/internal/ai/quota"
	"github.com/arturonavax/env/internal/platform"
	"github.com/arturonavax/env/internal/security"
)

func TestWaitForConnectivityAgainstLocalTCPListener(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	err = daemon.WaitForConnectivityWithOptions(context.Background(), daemon.ConnectivityOptions{
		Endpoints:   []string{ln.Addr().String()},
		DNSHost:     "localhost",
		Timeout:     2 * time.Second,
		Interval:    50 * time.Millisecond,
		DialTimeout: 250 * time.Millisecond,
		LookupHost: func(_ context.Context, _ string) ([]string, error) {
			return []string{"127.0.0.1"}, nil
		},
	})
	if err != nil {
		t.Fatalf("WaitForConnectivityWithOptions() unexpected error: %v", err)
	}

	// Verify fast timeout when unreachable
	closedLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closedAddr := closedLn.Addr().String()
	_ = closedLn.Close()

	err = daemon.WaitForConnectivityWithOptions(context.Background(), daemon.ConnectivityOptions{
		Endpoints:   []string{closedAddr},
		Timeout:     120 * time.Millisecond,
		Interval:    30 * time.Millisecond,
		DialTimeout: 30 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected connectivity error when TCP endpoint is closed")
	}
}

type mockCmdExec struct {
	failWake bool
}

func (m *mockCmdExec) RunWithInput(_ context.Context, _ security.RunOptions, binary string, args ...string) (string, error) {
	if m.failWake && binary == "agy" {
		return "", errors.New("503 upstream unavailable")
	}
	return "1", nil
}

func TestWakeCycleGuardrailAbortsSuspendOnProbeFailureAndSuspendsOnSuccess(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, aErr := ln.Accept()
			if aErr != nil {
				return
			}
			_ = c.Close()
		}
	}()

	connOpts := daemon.ConnectivityOptions{
		Endpoints:   []string{ln.Addr().String()},
		DNSHost:     "localhost",
		Timeout:     time.Second,
		Interval:    20 * time.Millisecond,
		DialTimeout: 200 * time.Millisecond,
		LookupHost:  func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil },
	}

	// Case 1: Failing wake probe MUST abort system suspend!
	t.Run("aborts suspend when wake probe fails", func(t *testing.T) {
		t.Parallel()

		home := t.TempDir()
		store := config.NewStateStore(filepath.Join(home, "state"))
		cfg, _ := config.DefaultConfig()
		cfg.Targets = []config.TargetConfig{
			{
				ID:            "gemini",
				Name:          "Gemini",
				Tool:          "agy",
				Enabled:       true,
				TelemetryType: "timed_fallback",
				WakeCommand:   []string{"agy", "-p", "Reply with: 1"},
			},
		}

		execMock := &mockCmdExec{failWake: true}
		qSvc := quota.NewService(cfg, store, quota.WithExecutor(execMock))

		var suspendCount atomic.Int32
		mgr := daemon.NewManager(
			cfg,
			store,
			qSvc,
			daemon.WithPlatform(platform.NewTestInfo(platform.TestInfoParams{OS: "linux", HomeDir: home})),
			daemon.WithCommandExecutor(execMock),
			daemon.WithLookPath(func(string) (string, error) { return "", errors.New("none") }),
			daemon.WithConnectivity(connOpts),
			daemon.WithSleepDelay(func(context.Context, time.Duration) error { return nil }),
			daemon.WithSuspendFunc(func(context.Context) (string, error) {
				suspendCount.Add(1)
				return "mock-suspend", nil
			}),
		)

		var out bytes.Buffer
		res, cycleErr := mgr.RunWakeCycle(context.Background(), &out)
		if cycleErr == nil {
			t.Fatal("expected RunWakeCycle error when wake probe fails")
		}
		if !res.AbortedSuspend || res.Suspended || suspendCount.Load() != 0 {
			t.Fatalf("expected suspend guardrail to abort suspend; got res=%+v suspendCount=%d", res, suspendCount.Load())
		}
	})

	// Case 2: Successful network + wake probe triggers safe suspend!
	t.Run("suspends cleanly after verified network and successful probe", func(t *testing.T) {
		t.Parallel()

		home := t.TempDir()
		store := config.NewStateStore(filepath.Join(home, "state"))
		cfg, _ := config.DefaultConfig()
		cfg.Targets = []config.TargetConfig{
			{
				ID:            "gemini",
				Name:          "Gemini",
				Tool:          "agy",
				Enabled:       true,
				TelemetryType: "timed_fallback",
				WakeCommand:   []string{"agy", "-p", "Reply with: 1"},
			},
		}

		execMock := &mockCmdExec{failWake: false}
		qSvc := quota.NewService(cfg, store, quota.WithExecutor(execMock))

		var suspendCount atomic.Int32
		mgr := daemon.NewManager(
			cfg,
			store,
			qSvc,
			daemon.WithPlatform(platform.NewTestInfo(platform.TestInfoParams{OS: "linux", HomeDir: home})),
			daemon.WithCommandExecutor(execMock),
			daemon.WithLookPath(func(string) (string, error) { return "", errors.New("none") }),
			daemon.WithConnectivity(connOpts),
			daemon.WithSleepDelay(func(context.Context, time.Duration) error { return nil }),
			daemon.WithSuspendFunc(func(context.Context) (string, error) {
				suspendCount.Add(1)
				return "systemctl suspend", nil
			}),
		)

		res, cycleErr := mgr.RunWakeCycle(context.Background(), &bytes.Buffer{})
		if cycleErr != nil {
			t.Fatalf("RunWakeCycle() unexpected error: %v", cycleErr)
		}
		if !res.Suspended || suspendCount.Load() != 1 {
			t.Fatalf("expected system suspend after full success; got res=%+v suspendCount=%d", res, suspendCount.Load())
		}
	})
}

func TestSystemdUnitAndLaunchdGenerationInTempDir(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Linux systemd units
	linuxHome := t.TempDir()
	cfg, _ := config.DefaultConfig()
	store := config.NewStateStore(filepath.Join(linuxHome, "state"))
	mgr := daemon.NewManager(
		cfg,
		store,
		nil,
		daemon.WithPlatform(platform.NewTestInfo(platform.TestInfoParams{OS: "linux", HomeDir: linuxHome})),
		daemon.WithBinaryPath(filepath.Join(linuxHome, ".local", "bin", "aitools")),
		daemon.WithLookPath(func(string) (string, error) { return "", errors.New("no systemctl in unit test") }),
	)

	inst, err := mgr.Install(ctx)
	if err != nil {
		t.Fatalf("Install() Linux error: %v", err)
	}
	if len(inst.Files) != 3 {
		t.Fatalf("expected 3 systemd unit files, got %v", inst.Files)
	}

	for _, unitName := range []string{daemon.UnitMonitorService, daemon.UnitWakeupService, daemon.UnitWakeupTimer} {
		p := filepath.Join(linuxHome, ".config", "systemd", "user", unitName)
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			t.Fatalf("expected unit %s to exist: %v", p, readErr)
		}
		if !strings.Contains(string(data), "[Unit]") {
			t.Errorf("unit %s missing [Unit] section:\n%s", unitName, string(data))
		}
	}

	if err := mgr.Uninstall(ctx); err != nil {
		t.Fatalf("Uninstall() Linux error: %v", err)
	}
	for _, unitName := range []string{daemon.UnitMonitorService, daemon.UnitWakeupService, daemon.UnitWakeupTimer} {
		p := filepath.Join(linuxHome, ".config", "systemd", "user", unitName)
		if _, statErr := os.Stat(p); !errors.Is(statErr, os.ErrNotExist) {
			t.Errorf("expected unit %s to be removed after Uninstall()", p)
		}
	}

	// macOS launchd plist
	macHome := t.TempDir()
	macMgr := daemon.NewManager(
		cfg,
		config.NewStateStore(filepath.Join(macHome, "state")),
		nil,
		daemon.WithPlatform(platform.NewTestInfo(platform.TestInfoParams{OS: "darwin", HomeDir: macHome})),
		daemon.WithLookPath(func(string) (string, error) { return "", errors.New("no launchctl in unit test") }),
	)
	macInst, err := macMgr.Install(ctx)
	if err != nil {
		t.Fatalf("Install() macOS error: %v", err)
	}
	if len(macInst.Files) != 1 || !strings.HasSuffix(macInst.Files[0], ".plist") {
		t.Fatalf("expected launchd plist file, got %v", macInst.Files)
	}
}
