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

type spyExecWithCommands struct {
	lastArgs []string
}

func (s *spyExecWithCommands) RunWithInput(_ context.Context, _ security.RunOptions, _ string, args ...string) (string, error) {
	s.lastArgs = args
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "/usage") {
		return `{
  "command": {
    "data": {
      "groups": [
        {
          "name": "Gemini Models",
          "buckets": [
            {"id": "gemini-5h", "name": "Five Hour Limit Remaining", "window": "5h", "remaining_fraction": 0.95, "reset_time": "2026-11-07T16:00:00Z"},
            {"id": "gemini-weekly", "name": "Weekly Limit Remaining", "window": "weekly", "remaining_fraction": 0.90, "reset_time": "2026-11-14T00:00:00Z"}
          ]
        },
        {
          "name": "Gemini 4 Argon",
          "buckets": [
            {"id": "gemini-argon-wtu-weekly", "name": "Gemini 4 Argon Weekly Limit", "window": "weekly", "remaining_fraction": 0.99, "reset_time": "2026-11-14T11:00:00Z"}
          ]
        }
      ]
    }
  }
}`, nil
	}
	return "1", nil
}

func TestAgyTargetGroupSubQuotaIsolationAndArgonWake(t *testing.T) {
	t.Parallel()

	spy := &spyExecWithCommands{}
	store := config.NewStateStore(t.TempDir())

	cfg := &config.Config{
		Targets: []config.TargetConfig{
			{
				ID:            "gemini",
				Name:          "Google Gemini",
				Tool:          "agy",
				Enabled:       true,
				TelemetryType: "agy_usage",
				BucketID:      "gemini-5h",
			},
			{
				ID:            "gemini-argon",
				Name:          "Gemini 4 Argon",
				Tool:          "agy",
				Enabled:       true,
				TelemetryType: "agy_usage",
				BucketID:      "gemini-argon-wtu-weekly",
			},
		},
	}

	svc := quota.NewService(
		cfg,
		store,
		quota.WithExecutor(spy),
		quota.WithClock(func() time.Time { return time.Date(2026, 11, 7, 11, 0, 0, 0, time.UTC) }),
	)

	ctx := context.Background()

	// 1. Evaluate both targets
	evals, err := svc.EvaluateAll(ctx, "")
	if err != nil {
		t.Fatalf("EvaluateAll: %v", err)
	}
	if len(evals) != 2 {
		t.Fatalf("expected 2 evals, got %d", len(evals))
	}

	// Target 1: gemini should only show "Gemini Models" buckets (len = 2)
	if len(evals[0].SubQuotas) != 2 {
		t.Fatalf("expected 2 subquotas in gemini target, got %d", len(evals[0].SubQuotas))
	}
	for _, sq := range evals[0].SubQuotas {
		if sq.Group != "Gemini Models" {
			t.Errorf("leaked group %q into gemini target", sq.Group)
		}
	}

	// Target 2: gemini-argon should only show "Gemini 4 Argon" buckets (len = 1)
	if len(evals[1].SubQuotas) != 1 {
		t.Fatalf("expected 1 subquota in argon target, got %d", len(evals[1].SubQuotas))
	}
	if evals[1].SubQuotas[0].Group != "Gemini 4 Argon" {
		t.Errorf("unexpected group in argon target: %q", evals[1].SubQuotas[0].Group)
	}

	// 2. A running window is NEVER probed, not even with force.
	spy.lastArgs = nil
	rep, err := svc.Wake(ctx, true, "gemini-argon")
	if err != nil {
		t.Fatalf("Wake gemini-argon: %v", err)
	}
	if len(rep.Woken) != 0 || len(rep.Skipped) != 1 || rep.Skipped[0] != "gemini-argon" {
		t.Fatalf("expected active gemini-argon skipped, got: %+v", rep)
	}
	for _, a := range spy.lastArgs {
		if a == "Reply with: 1" {
			t.Fatalf("wake probe must not be sent to an active window, args: %v", spy.lastArgs)
		}
	}
}

type spyExpired3p struct{ probes [][]string }

func (s *spyExpired3p) RunWithInput(_ context.Context, _ security.RunOptions, _ string, args ...string) (string, error) {
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "/usage") {
		return `{"command":{"data":{"groups":[
{"name":"Gemini Models","buckets":[{"id":"gemini-5h","window":"5h","remaining_fraction":0.9,"reset_time":"2026-11-07T16:00:00Z"}]},
{"name":"Claude and GPT models","buckets":[{"id":"3p-5h","window":"5h","remaining_fraction":1,"reset_time":"2026-11-07T09:00:00Z"}]}
]}}}`, nil
	}
	s.probes = append(s.probes, args)
	return "1", nil
}

func TestWakeClaudeGPTWhenExpiredAndSkipGeminiWhenActive(t *testing.T) {
	t.Parallel()

	spy := &spyExpired3p{}
	cfg := &config.Config{Targets: []config.TargetConfig{
		{ID: "gemini", Tool: "agy", Enabled: true, TelemetryType: "agy_usage", BucketID: "gemini-5h"},
		{ID: "claude-gpt", Tool: "agy", Enabled: true, TelemetryType: "agy_usage", BucketID: "3p-5h"},
	}}
	svc := quota.NewService(cfg, config.NewStateStore(t.TempDir()), quota.WithExecutor(spy),
		quota.WithClock(func() time.Time { return time.Date(2026, 11, 7, 11, 0, 0, 0, time.UTC) }))

	rep, err := svc.Wake(context.Background(), false, "")
	if err != nil {
		t.Fatalf("Wake: %v", err)
	}
	if len(rep.Woken) != 1 || rep.Woken[0] != "claude-gpt" {
		t.Fatalf("expected only claude-gpt woken, got %+v", rep)
	}
	if len(spy.probes) != 1 || !strings.Contains(strings.Join(spy.probes[0], " "), "Claude Sonnet 5.5 (Low)") {
		t.Fatalf("expected one Claude probe, got %v", spy.probes)
	}
}
