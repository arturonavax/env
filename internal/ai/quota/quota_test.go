package quota_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/ai/quota"
	"github.com/arturonavax/env/internal/security"
)

type fakeQuotaExec struct {
	wakeCalls int
	failWake  bool
}

func (f *fakeQuotaExec) RunWithInput(_ context.Context, _ security.RunOptions, binary string, args ...string) (string, error) {
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "/usage") {
		return "", errors.New("telemetry unavailable, use timed_fallback")
	}
	f.wakeCalls++
	if f.failWake {
		return "", errors.New("rate limited")
	}
	return "1", nil
}

func TestRolling5HourWindowMathAndWakeTransitions(t *testing.T) {
	t.Parallel()

	cfg, err := config.DefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Targets = []config.TargetConfig{
		{
			ID:                    "gemini",
			Name:                  "Gemini 5h",
			Tool:                  "agy",
			Account:               "default",
			Enabled:               true,
			TelemetryType:         "timed_fallback",
			FallbackIntervalHours: 5,
			WakeCommand:           []string{"agy", "-p", "Reply with: 1"},
		},
	}

	store := config.NewStateStore(t.TempDir())
	execMock := &fakeQuotaExec{}

	currTime := time.Date(2026, 10, 29, 4, 0, 0, 0, time.UTC)
	svc := quota.NewService(
		cfg,
		store,
		quota.WithExecutor(execMock),
		quota.WithClock(func() time.Time { return currTime }),
		quota.WithNetworkChecker(func(context.Context) error { return nil }),
	)

	ctx := context.Background()

	// 1. Initially unprimed -> idle & NeedsWake
	evals, err := svc.EvaluateAll(ctx, "")
	if err != nil {
		t.Fatalf("EvaluateAll() initial error: %v", err)
	}
	if len(evals) != 1 || evals[0].State != quota.StateIdle || !evals[0].NeedsWake {
		t.Fatalf("initial eval = %+v, want state=idle needs_wake=true", evals)
	}

	// 2. Wake primes window
	rep, err := svc.Wake(ctx, false, "")
	if err != nil {
		t.Fatalf("Wake() error: %v", err)
	}
	if len(rep.Woken) != 1 || execMock.wakeCalls != 1 {
		t.Fatalf("Wake report = %+v, wakeCalls=%d, want 1 woken", rep, execMock.wakeCalls)
	}

	// 3. Advance 2 hours -> active with 3h (10800s) remaining; non-forced Wake skips
	currTime = currTime.Add(2 * time.Hour)
	evals, err = svc.EvaluateAll(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if evals[0].State != quota.StateActive || evals[0].RemainingSeconds != 3*3600 || evals[0].NeedsWake {
		t.Fatalf("at +2h eval = %+v, want active with 10800s left", evals[0])
	}

	rep2, err := svc.Wake(ctx, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rep2.Skipped) != 1 || execMock.wakeCalls != 1 {
		t.Errorf("expected active target to be skipped without probing again: %+v", rep2)
	}

	// 4. Advance past 5-hour boundary (+3h 1m) -> expired & NeedsWake
	currTime = currTime.Add(3*time.Hour + time.Minute)
	evals, err = svc.EvaluateAll(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if evals[0].State != quota.StateExpired || !evals[0].NeedsWake || evals[0].RemainingSeconds != 0 {
		t.Fatalf("at +5h01m eval = %+v, want expired & needs_wake", evals[0])
	}
}

func TestWatchModelUpdateAndNonTTYSnapshot(t *testing.T) {
	t.Parallel()

	cfg, _ := config.DefaultConfig()
	store := config.NewStateStore(t.TempDir())
	svc := quota.NewService(cfg, store, quota.WithExecutor(&fakeQuotaExec{}))

	var snap bytes.Buffer
	if err := quota.RunWatch(context.Background(), &snap, svc, 5*time.Second, "", false); err != nil {
		t.Fatalf("RunWatch non-TTY error: %v", err)
	}
	if !strings.Contains(snap.String(), "gemini") {
		t.Errorf("expected gemini target in non-TTY watch snapshot:\n%s", snap.String())
	}

	m := quota.NewWatchModel(context.Background(), svc, 10*time.Second, "")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 110, Height: 28})
	wm := updated.(quota.WatchModel)
	v := wm.View()
	if v.Content == "" {
		t.Error("expected non-empty Bubble Tea v2 View content")
	}
}
