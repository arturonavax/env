package quota

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
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

// SubQuota represents a granular model group or window limit (5h, weekly).
type SubQuota struct {
	ID               string    `json:"id"`
	Group            string    `json:"group"`
	Name             string    `json:"name"`
	Window           string    `json:"window"`
	RemainingPercent float64   `json:"remaining_percent"`
	ResetTimeLocal   string    `json:"reset_time_local"`
	ResetAt          time.Time `json:"reset_at,omitempty"`
	Disabled         bool      `json:"disabled,omitempty"`
	Description      string    `json:"description,omitempty"`
}

// TargetEvaluation represents the evaluated rolling quota window state of a target.
type TargetEvaluation struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Tool             string     `json:"tool"`
	Account          string     `json:"account"`
	Enabled          bool       `json:"enabled"`
	TelemetryType    string     `json:"telemetry_type"`
	State            string     `json:"state"`
	RemainingPercent float64    `json:"remaining_percent"`
	RemainingSeconds int        `json:"remaining_seconds"`
	ResetTimeLocal   string     `json:"reset_time_local"`
	ResetAt          time.Time  `json:"reset_at,omitempty"`
	IsActive         bool       `json:"is_active"`
	StatusLabel      string     `json:"status_label"`
	NeedsWake        bool       `json:"needs_wake"`
	SubQuotas        []SubQuota `json:"sub_quotas,omitempty"`
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
				Name        string `json:"name"`
				Description string `json:"description"`
				Buckets     []struct {
					ID                string  `json:"id"`
					Name              string  `json:"name"`
					Window            string  `json:"window"`
					Description       string  `json:"description"`
					Disabled          bool    `json:"disabled"`
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

	var primaryFound bool
	eval.SubQuotas = make([]SubQuota, 0, 8)

	// Identify target group if bucket_id is specified
	targetGroup := ""
	if target.BucketID != "" {
		for _, g := range env.Command.Data.Groups {
			for _, b := range g.Buckets {
				if b.ID == target.BucketID {
					targetGroup = g.Name
					break
				}
			}
			if targetGroup != "" {
				break
			}
		}
	}

	for _, g := range env.Command.Data.Groups {
		if targetGroup != "" && g.Name != targetGroup {
			continue
		}
		for _, b := range g.Buckets {
			remPct := math.Round(b.RemainingFraction*1000) / 10
			sq := SubQuota{
				ID:               b.ID,
				Group:            g.Name,
				Name:             b.Name,
				Window:           b.Window,
				RemainingPercent: remPct,
				Disabled:         b.Disabled,
				Description:      b.Description,
			}

			var resetAt time.Time
			var hasReset bool
			if strings.TrimSpace(b.ResetTime) != "" {
				if t, parseErr := parseResetTimestamp(b.ResetTime); parseErr == nil {
					resetAt = t
					hasReset = true
					sq.ResetAt = t
					sq.ResetTimeLocal = t.Local().Format("15:04:05")
				}
			}
			eval.SubQuotas = append(eval.SubQuotas, sq)

			isTargetBucket := (target.BucketID != "" && b.ID == target.BucketID)
			isFallback5h := (target.BucketID == "" && !primaryFound && (b.Window == "5h" || b.ID == "gemini-5h"))

			if isTargetBucket || isFallback5h {
				primaryFound = true
				eval.RemainingPercent = remPct

				if !hasReset || strings.TrimSpace(b.ResetTime) == "" {
					eval.State = StateIdle
					eval.IsActive = false
					eval.NeedsWake = true
					eval.RemainingSeconds = 0
					eval.StatusLabel = "IDLE (no rolling timer active)"
				} else {
					eval.ResetAt = resetAt
					eval.ResetTimeLocal = resetAt.Local().Format("15:04:05")

					if resetAt.After(now) {
						rem := resetAt.Sub(now)
						eval.State = StateActive
						eval.IsActive = true
						eval.NeedsWake = false
						eval.RemainingSeconds = int(rem.Seconds())
						if rem.Hours() > 24 {
							d := int(rem.Hours()) / 24
							h := int(rem.Hours()) % 24
							eval.StatusLabel = fmt.Sprintf("ACTIVE (~%dd %dh left, resets at %s)", d, h, resetAt.Local().Format("Jan 02 15:04"))
						} else {
							h := int(rem.Hours())
							m := int(rem.Minutes()) % 60
							eval.StatusLabel = fmt.Sprintf("ACTIVE (%dh %02dm left, resets at %s)", h, m, resetAt.Local().Format("15:04"))
						}
					} else {
						eval.State = StateExpired
						eval.IsActive = false
						eval.NeedsWake = true
						eval.RemainingSeconds = 0
						eval.StatusLabel = fmt.Sprintf("EXPIRED (refreshed at %s)", resetAt.Local().Format("15:04"))
					}
				}
			}
		}
	}

	if len(eval.SubQuotas) > 0 {
		if !primaryFound {
			eval.RemainingPercent = eval.SubQuotas[0].RemainingPercent
			eval.ResetAt = eval.SubQuotas[0].ResetAt
			eval.ResetTimeLocal = eval.SubQuotas[0].ResetTimeLocal
			eval.State = StateActive
			eval.IsActive = true
			eval.StatusLabel = "ACTIVE"
		}
		return true
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
			if tool == "agy" {
				model := "gemini-3.8-flash-low"
				if strings.Contains(strings.ToLower(target.ID), "argon") || strings.Contains(strings.ToLower(target.BucketID), "argon") {
					model = "gemini-4-argon-low"
				}
				wakeCmd = []string{tool, "-p", "Reply with: 1", "--model", model, "--effort", "low", "--disable-slash-commands"}
			} else {
				wakeCmd = []string{tool, "-p", "Reply with: 1"}
			}
		} else if wakeCmd[0] == "agy" {
			hasDisableSlash := false
			for _, arg := range wakeCmd {
				if arg == "--disable-slash-commands" {
					hasDisableSlash = true
					break
				}
			}
			if !hasDisableSlash {
				wakeCmd = append(wakeCmd, "--disable-slash-commands")
			}
		}

		timeout := time.Duration(target.TimeoutSec) * time.Second
		if timeout <= 0 {
			timeout = 25 * time.Second
		}

		_ = s.store.AppendLog("INFO", "PROBE_DISPATCH", fmt.Sprintf("Sending wakeup probe to %q", target.ID), nil)
		runOpts := security.RunOptions{
			Dir:     os.TempDir(),
			Timeout: timeout,
		}
		probeOut, probeErr := s.exec.RunWithInput(ctx, runOpts, wakeCmd[0], wakeCmd[1:]...)
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

// FormatTable renders evaluations as an aligned terminal table with Tokyo Night styling,
// showing both target-level and detailed sub-quota breakdowns (5-hour and weekly limits).
func FormatTable(evals []TargetEvaluation) string {
	var sb strings.Builder
	const (
		cReset   = "\033[0m"
		cBold    = "\033[1m"
		cDim     = "\033[2m"
		cRed     = "\033[38;2;247;118;142m" // Tokyo Night Red (#f7768e)
		cBlue    = "\033[38;2;122;162;247m" // Tokyo Night Blue (#7aa2f7)
		cCyan    = "\033[38;2;125;207;255m" // Tokyo Night Cyan (#7dcfff)
		cGreen   = "\033[38;2;158;206;106m" // Tokyo Night Green (#9ece6a)
		cYellow  = "\033[38;2;224;175;104m" // Tokyo Night Yellow (#e0af68)
		cMuted   = "\033[38;2;86;95;137m"   // Tokyo Night Gray (#565f89)
		cWhite   = "\033[38;2;192;202;245m" // Tokyo Night Foreground (#c0caf5)
	)

	sb.WriteString(fmt.Sprintf("%s%sANX AI%s %s·%s %sModels Rolling Quota & Reset Timers%s\n", cBold, cRed, cReset, cMuted, cReset, cBold, cReset))
	sb.WriteString(fmt.Sprintf("%s%s%s\n", cMuted, strings.Repeat("━", 88), cReset))

	if len(evals) == 0 {
		sb.WriteString(fmt.Sprintf("%s(no targets configured)%s\n", cMuted, cReset))
		return sb.String()
	}

	for _, e := range evals {
		statusColor := cGreen
		if e.State == StateExpired {
			statusColor = cRed
		} else if e.State == StateIdle {
			statusColor = cYellow
		} else if !e.Enabled {
			statusColor = cMuted
		}

		fmt.Fprintf(
			&sb,
			"%s%s● %s%s %s[%s / %s / %s]%s %s(%s)%s\n",
			cBold,
			statusColor,
			e.Name,
			cReset,
			cMuted,
			e.ID,
			e.Tool,
			e.Account,
			cReset,
			cDim,
			strings.ToUpper(e.State),
			cReset,
		)

		if len(e.SubQuotas) > 0 {
			currentGroup := ""
			for _, sq := range e.SubQuotas {
				if sq.Group != currentGroup {
					currentGroup = sq.Group
					fmt.Fprintf(&sb, "  %s%s%s%s\n", cBold, cCyan, currentGroup, cReset)
				}
				bar := FormatProgressBar(sq.RemainingPercent, 10)
				resetInfo := sq.ResetTimeLocal
				if sq.Disabled {
					resetInfo = fmt.Sprintf("%s(disabled)%s", cMuted, cReset)
					bar = "──────────   -  "
				} else if !sq.ResetAt.IsZero() {
					now := time.Now()
					if sq.ResetAt.After(now) {
						delta := sq.ResetAt.Sub(now)
						if delta.Hours() > 24 {
							d := int(delta.Hours()) / 24
							h := int(delta.Hours()) % 24
							resetInfo = fmt.Sprintf("%s (~%dd %dh left)", sq.ResetAt.Local().Format("Jan 02 15:04"), d, h)
						} else {
							h := int(delta.Hours())
							m := int(delta.Minutes()) % 60
							resetInfo = fmt.Sprintf("%s (~%dh %02dm left)", sq.ResetTimeLocal, h, m)
						}
					} else {
						resetInfo = fmt.Sprintf("%s (window ended)", sq.ResetTimeLocal)
					}
				}

				nameLabel := sq.Name
				if sq.Window == "5h" {
					nameLabel = "5-Hour Window"
				} else if sq.Window == "weekly" {
					nameLabel = "Weekly Window"
				}

				fmt.Fprintf(
					&sb,
					"    %s%-16s%s %s%s%s  %sResets:%s %-28s\n",
					cMuted,
					nameLabel,
					cReset,
					cWhite,
					bar,
					cReset,
					cMuted,
					cReset,
					resetInfo,
				)
			}
		} else {
			bar := FormatProgressBar(e.RemainingPercent, 10)
			if !e.Enabled {
				bar = "──────────   -  "
			}
			fmt.Fprintf(
				&sb,
				"    %s%-16s%s %s%s%s  %sResets:%s %-28s %s\n",
				cMuted,
				"Rolling Window",
				cReset,
				cWhite,
				bar,
				cReset,
				cMuted,
				cReset,
				e.ResetTimeLocal,
				e.StatusLabel,
			)
		}
		sb.WriteString(fmt.Sprintf("%s%s%s\n", cMuted, strings.Repeat("─", 88), cReset))
	}
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
