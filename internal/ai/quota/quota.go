package quota

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/security"
)

const (
	StateActive   = "active"
	StateExpired  = "expired"
	StateIdle     = "idle"
	StateDisabled = "disabled"
)

// TargetEvaluation represents the evaluated rolling quota window state of a target.
type TargetEvaluation struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Tool             string    `json:"tool"`
	Account          string    `json:"account"`
	Enabled          bool      `json:"enabled"`
	TelemetryType    string    `json:"telemetry_type"`
	State            string    `json:"state"`
	RemainingPercent float64   `json:"remaining_percent"`
	RemainingSeconds int       `json:"remaining_seconds"`
	ResetTimeLocal   string    `json:"reset_time_local"`
	ResetAt          time.Time `json:"reset_at,omitempty"`
	IsActive         bool      `json:"is_active"`
	StatusLabel      string    `json:"status_label"`
	NeedsWake        bool      `json:"needs_wake"`
}

// WakeOptions configures a wakeup priming run.
type WakeOptions struct {
	Force        bool
	TargetFilter string
	Reason       string
	Out          io.Writer
}

// WakeReport summarizes the outcome of a Wake execution across targets.
type WakeReport struct {
	Woken   []string          `json:"woken"`
	Skipped []string          `json:"skipped"`
	Failed  map[string]string `json:"failed"`
}

// AllSucceeded reports whether zero targets failed during the wake cycle.
func (r *WakeReport) AllSucceeded() bool {
	return r != nil && len(r.Failed) == 0
}

// CommandExecutor runs status and wake commands (satisfied by *security.Runner or test doubles).
type CommandExecutor interface {
	RunWithInput(ctx context.Context, opts security.RunOptions, binary string, args ...string) (string, error)
}

// Service evaluates rolling 5-hour quota windows and dispatches wakeup probes.
type Service struct {
	cfg          *config.Config
	store        *config.StateStore
	exec         CommandExecutor
	checkNetwork func(context.Context) error
	nowFunc      func() time.Time
}

// ServiceOption customizes a quota Service.
type ServiceOption func(*Service)

// WithExecutor injects a custom CommandExecutor for status and wake probes.
func WithExecutor(exec CommandExecutor) ServiceOption {
	return func(s *Service) {
		if exec != nil {
			s.exec = exec
		}
	}
}

// WithNetworkChecker injects a network connectivity verifier executed before Wake probes.
func WithNetworkChecker(fn func(context.Context) error) ServiceOption {
	return func(s *Service) {
		s.checkNetwork = fn
	}
}

// WithClock overrides time.Now for deterministic rolling window unit tests.
func WithClock(fn func() time.Time) ServiceOption {
	return func(s *Service) {
		if fn != nil {
			s.nowFunc = fn
		}
	}
}

// NewService constructs a quota Service.
func NewService(cfg *config.Config, store *config.StateStore, opts ...ServiceOption) *Service {
	if cfg == nil {
		cfg, _ = config.DefaultConfig()
	}
	if store == nil {
		store = config.NewStateStore("")
	}
	svc := &Service{
		cfg:     cfg,
		store:   store,
		exec:    security.NewRunner(25 * time.Second),
		nowFunc: time.Now,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// EvaluateAll evaluates all configured targets (optionally filtered by targetFilter).
func (s *Service) EvaluateAll(ctx context.Context, targetFilter string) ([]TargetEvaluation, error) {
	st, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	now := s.nowFunc().UTC()
	evals := make([]TargetEvaluation, 0, len(s.cfg.Targets))
	for _, target := range s.cfg.Targets {
		if targetFilter != "" && target.ID != targetFilter {
			continue
		}
		evals = append(evals, s.EvaluateTargetAt(ctx, target, st, now))
	}
	return evals, nil
}

// EvaluateTargetAt computes the rolling quota window status for a single target at timestamp now.
func (s *Service) EvaluateTargetAt(
	ctx context.Context,
	target config.TargetConfig,
	st *config.State,
	now time.Time,
) TargetEvaluation {
	name := target.Name
	if name == "" {
		name = target.ID
	}
	account := target.Account
	if account == "" {
		account = "default"
	}
	tType := target.TelemetryType
	if tType == "" {
		tType = "timed_fallback"
	}

	eval := TargetEvaluation{
		ID:               target.ID,
		Name:             name,
		Tool:             target.Tool,
		Account:          account,
		Enabled:          target.Enabled,
		TelemetryType:    tType,
		State:            StateIdle,
		RemainingPercent: 100.0,
		ResetTimeLocal:   "-",
		StatusLabel:      "IDLE (ready to prime)",
		NeedsWake:        true,
	}

	if !target.Enabled {
		eval.State = StateDisabled
		eval.StatusLabel = "DISABLED"
		eval.NeedsWake = false
		return eval
	}

	switch tType {
	case "agy_usage":
		if ok := s.applyAgyUsageTelemetry(ctx, target, now, &eval); ok {
			return eval
		}
	case "claude_status":
		if ok := s.applyClaudeTelemetry(ctx, target, &eval); ok {
			return eval
		}
	}

	s.applyTimedFallback(target, st, now, &eval)
	return eval
}

type agyUsageEnvelope struct {
	Command struct {
		Data struct {
			Groups []struct {
				Name    string `json:"name"`
				Buckets []struct {
					ID                string  `json:"id"`
					Name              string  `json:"name"`
					RemainingFraction float64 `json:"remaining_fraction"`
					ResetTime         string  `json:"reset_time"`
				} `json:"buckets"`
			} `json:"groups"`
		} `json:"data"`
	} `json:"command"`
}

func (s *Service) applyAgyUsageTelemetry(
	ctx context.Context,
	target config.TargetConfig,
	now time.Time,
	eval *TargetEvaluation,
) bool {
	cmd := target.StatusCommand
	if len(cmd) == 0 {
		cmd = []string{"agy", "-p", "/usage", "--output-format", "json"}
	}
	timeout := time.Duration(target.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	out, err := s.exec.RunWithInput(ctx, security.RunOptions{Timeout: timeout}, cmd[0], cmd[1:]...)
	if err != nil || strings.TrimSpace(out) == "" {
		return false
	}

	var env agyUsageEnvelope
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); err != nil {
		return false
	}

	for _, g := range env.Command.Data.Groups {
		for _, b := range g.Buckets {
			if target.BucketID != "" && b.ID != target.BucketID {
				continue
			}
			eval.RemainingPercent = math.Round(b.RemainingFraction*1000) / 10
			if strings.TrimSpace(b.ResetTime) == "" {
				eval.State = StateIdle
				eval.IsActive = false
				eval.NeedsWake = true
				eval.StatusLabel = "IDLE (no rolling timer active)"
				return true
			}

			resetAt, parseErr := parseResetTimestamp(b.ResetTime)
			if parseErr != nil {
				return false
			}
			eval.ResetAt = resetAt
			eval.ResetTimeLocal = resetAt.Local().Format("15:04:05")

			if resetAt.After(now) {
				rem := resetAt.Sub(now)
				eval.State = StateActive
				eval.IsActive = true
				eval.NeedsWake = false
				eval.RemainingSeconds = int(rem.Seconds())
				h := int(rem.Hours())
				m := int(rem.Minutes()) % 60
				eval.StatusLabel = fmt.Sprintf("ACTIVE (%dh %02dm left, resets at %s)", h, m, resetAt.Local().Format("15:04"))
			} else {
				eval.State = StateExpired
				eval.IsActive = false
				eval.NeedsWake = true
				eval.RemainingSeconds = 0
				eval.StatusLabel = fmt.Sprintf("EXPIRED (refreshed at %s)", resetAt.Local().Format("15:04"))
			}
			return true
		}
	}
	return false
}

func (s *Service) applyClaudeTelemetry(
	ctx context.Context,
	target config.TargetConfig,
	eval *TargetEvaluation,
) bool {
	cmd := target.StatusCommand
	if len(cmd) == 0 {
		cmd = []string{"claude", "status", "--json"}
	}
	out, err := s.exec.RunWithInput(ctx, security.RunOptions{Timeout: 15 * time.Second}, cmd[0], cmd[1:]...)
	if err != nil || strings.TrimSpace(out) == "" {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &payload); err != nil {
		return false
	}
	if rem, ok := payload["remaining_percent"].(float64); ok {
		eval.RemainingPercent = rem
	}
	eval.State = StateActive
	eval.IsActive = true
	eval.NeedsWake = false
	eval.StatusLabel = "ACTIVE (claude status verified)"
	return true
}

func (s *Service) applyTimedFallback(
	target config.TargetConfig,
	st *config.State,
	now time.Time,
	eval *TargetEvaluation,
) {
	hours := target.FallbackIntervalHours
	if hours <= 0 {
		hours = 5
	}
	windowDur := time.Duration(hours) * time.Hour
	lastWake := st.LastWakeTime(target.ID)

	if lastWake.IsZero() {
		eval.State = StateIdle
		eval.IsActive = false
		eval.NeedsWake = true
		eval.RemainingSeconds = 0
		eval.StatusLabel = "IDLE (unprimed, ready to prime)"
		return
	}

	resetAt := lastWake.Add(windowDur)
	eval.ResetAt = resetAt
	eval.ResetTimeLocal = resetAt.Local().Format("15:04:05")

	if now.Before(resetAt) {
		rem := resetAt.Sub(now)
		eval.State = StateActive
		eval.IsActive = true
		eval.NeedsWake = false
		eval.RemainingSeconds = int(rem.Seconds())
		h := int(rem.Hours())
		m := int(rem.Minutes()) % 60
		eval.StatusLabel = fmt.Sprintf("ACTIVE (%dh %02dm left, resets ~%s)", h, m, resetAt.Local().Format("15:04"))
		if ts, ok := st.Targets[target.ID]; ok && ts.RemainingPercent > 0 {
			eval.RemainingPercent = ts.RemainingPercent
		}
		return
	}

	eval.State = StateExpired
	eval.IsActive = false
	eval.NeedsWake = true
	eval.RemainingSeconds = 0
	eval.StatusLabel = fmt.Sprintf("EXPIRED (window ended at %s)", resetAt.Local().Format("15:04"))
}

func parseResetTimestamp(raw string) (time.Time, error) {
	s := strings.TrimSpace(raw)
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp %q", raw)
}

// Wake evaluates configured targets and sends a minimal wakeup probe ("Reply with: 1") to expired/idle targets.
func (s *Service) Wake(ctx context.Context, force bool, targetFilter string) (*WakeReport, error) {
	return s.WakeWithOptions(ctx, WakeOptions{
		Force:        force,
		TargetFilter: targetFilter,
		Reason:       "manual",
	})
}

// WakeWithOptions executes a full wakeup priming pass with structured logging and state updates.
func (s *Service) WakeWithOptions(ctx context.Context, opts WakeOptions) (*WakeReport, error) {
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	reason := opts.Reason
	if reason == "" {
		reason = "manual"
	}

	_ = s.store.AppendLog("INFO", "WAKE_CYCLE_START", fmt.Sprintf("Initiating wakeup cycle (reason=%s, force=%v)", reason, opts.Force), nil)

	if s.checkNetwork != nil {
		fmt.Fprintln(out, "Verifying network connectivity...")
		if err := s.checkNetwork(ctx); err != nil {
			_ = s.store.AppendLog("ERROR", "NETWORK_TIMEOUT", err.Error(), nil)
			return nil, fmt.Errorf("network connectivity verification failed: %w", err)
		}
	}

	st, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	now := s.nowFunc().UTC()
	report := &WakeReport{
		Woken:   []string{},
		Skipped: []string{},
		Failed:  map[string]string{},
	}

	for _, target := range s.cfg.Targets {
		if opts.TargetFilter != "" && target.ID != opts.TargetFilter {
			continue
		}
		if !target.Enabled {
			continue
		}

		eval := s.EvaluateTargetAt(ctx, target, st, now)
		fmt.Fprintf(out, "Target %s [%s/%s]: %s (%.1f%%)\n", eval.Name, eval.Tool, eval.Account, eval.StatusLabel, eval.RemainingPercent)

		if !opts.Force && !eval.NeedsWake {
			report.Skipped = append(report.Skipped, target.ID)
			_ = s.store.AppendLog("INFO", "PROBE_SKIPPED", fmt.Sprintf("Target %q window active", target.ID), nil)
			continue
		}

		wakeCmd := target.WakeCommand
		if len(wakeCmd) == 0 {
			tool := target.Tool
			if tool == "" {
				tool = "agy"
			}
			wakeCmd = []string{tool, "-p", "Reply with: 1"}
		}

		timeout := time.Duration(target.TimeoutSec) * time.Second
		if timeout <= 0 {
			timeout = 25 * time.Second
		}

		_ = s.store.AppendLog("INFO", "PROBE_DISPATCH", fmt.Sprintf("Sending wakeup probe to %q", target.ID), nil)
		probeOut, probeErr := s.exec.RunWithInput(ctx, security.RunOptions{Timeout: timeout}, wakeCmd[0], wakeCmd[1:]...)
		if probeErr != nil {
			errMsg := probeErr.Error()
			report.Failed[target.ID] = errMsg
			st.RecordWakeFailure(target.ID, errMsg)
			_ = s.store.AppendLog("ERROR", "PROBE_FAILED", fmt.Sprintf("Target %q failed: %s", target.ID, errMsg), nil)
			fmt.Fprintf(out, "  -> FAILED: %s\n", errMsg)
			continue
		}

		sample := strings.TrimSpace(strings.ReplaceAll(probeOut, "\n", " "))
		if len(sample) > 60 {
			sample = sample[:60]
		}
		report.Woken = append(report.Woken, target.ID)
		st.RecordWakeSuccess(target.ID, s.nowFunc().UTC(), eval.RemainingPercent)
		_ = s.store.AppendLog("INFO", "PROBE_SUCCESS", fmt.Sprintf("Target %q woken (response=%q)", target.ID, sample), nil)
		fmt.Fprintf(out, "  -> WOKEN (%s)\n", sample)
	}

	st.LastWakeEpoch = s.nowFunc().UTC().Unix()
	if saveErr := s.store.Save(st); saveErr != nil {
		return report, saveErr
	}

	if len(report.Failed) > 0 {
		_ = s.store.AppendLog("ERROR", "WAKE_CYCLE_ERROR", fmt.Sprintf("%d target probes failed", len(report.Failed)), nil)
		return report, errors.New("one or more target wakeup probes failed")
	}

	_ = s.store.AppendLog("INFO", "WAKE_CYCLE_SUCCESS", fmt.Sprintf("Wake cycle completed (woken=%d, skipped=%d)", len(report.Woken), len(report.Skipped)), nil)
	return report, nil
}

// FormatJSON renders evaluations as indented JSON.
func FormatJSON(evals []TargetEvaluation) (string, error) {
	if evals == nil {
		evals = []TargetEvaluation{}
	}
	b, err := json.MarshalIndent(evals, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FormatTable renders evaluations as an aligned terminal table with ASCII/Unicode progress bars.
func FormatTable(evals []TargetEvaluation) string {
	var sb strings.Builder
	sb.WriteString("ANX AI · Models Rolling Quota & Reset Timers\n")
	sb.WriteString(strings.Repeat("─", 86) + "\n")
	fmt.Fprintf(&sb, "%-14s %-8s %-10s %-19s %-10s %s\n", "ID", "TOOL", "ACCOUNT", "QUOTA", "RESETS AT", "STATUS")
	sb.WriteString(strings.Repeat("─", 86) + "\n")

	if len(evals) == 0 {
		sb.WriteString("(no targets configured)\n")
		return sb.String()
	}

	for _, e := range evals {
		bar := FormatProgressBar(e.RemainingPercent, 10)
		if !e.Enabled {
			bar = "──────────   -  "
		}
		fmt.Fprintf(
			&sb,
			"%-14s %-8s %-10s %-19s %-10s %s\n",
			e.ID,
			e.Tool,
			e.Account,
			bar,
			e.ResetTimeLocal,
			strings.ToUpper(e.State)+" · "+e.StatusLabel,
		)
	}
	sb.WriteString(strings.Repeat("─", 86))
	return sb.String()
}

// FormatProgressBar renders a fixed-width Unicode block progress bar plus percentage.
func FormatProgressBar(percent float64, width int) string {
	if width <= 0 {
		width = 10
	}
	clamped := math.Max(0, math.Min(100, percent))
	filled := int(math.Round((clamped / 100.0) * float64(width)))
	empty := width - filled
	return fmt.Sprintf("%s%s %5.1f%%", strings.Repeat("█", filled), strings.Repeat("░", empty), clamped)
}
