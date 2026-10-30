package config

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxDaemonLogBytes = 5 * 1024 * 1024 // 5 MB log rotation threshold

// TargetWindowState tracks quota window timestamps, snapshots, and failure counters per target.
type TargetWindowState struct {
	WindowStartedAt     time.Time `json:"window_started_at,omitempty"`
	LastWakeAt          time.Time `json:"last_wake_at,omitempty"`
	ResetAt             time.Time `json:"reset_at,omitempty"`
	RemainingPercent    float64   `json:"remaining_percent"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastError           string    `json:"last_error,omitempty"`
}

// State persists runtime daemon schedules, rolling quota windows, and wake timestamps.
type State struct {
	ScheduledWakeEpoch int64                        `json:"scheduled_wake_epoch"`
	ExpectingAutoWake  bool                         `json:"expecting_auto_wake"`
	AutoWakeEnabled    *bool                        `json:"auto_wake_enabled,omitempty"`
	LastWakeEpoch      int64                        `json:"last_wake_epoch"`
	LastAutoWakeEpoch  int64                        `json:"last_auto_wake_epoch"`
	TargetsLastWake    map[string]int64             `json:"targets_last_wake"`
	Targets            map[string]TargetWindowState `json:"targets"`
	NotifiedResets     map[string]int64             `json:"notified_resets"`
}

func newDefaultState() *State {
	return &State{
		TargetsLastWake: map[string]int64{},
		Targets:         map[string]TargetWindowState{},
		NotifiedResets:  map[string]int64{},
	}
}

// UnmarshalJSON provides backward compatibility for legacy Python float timestamps and json.Number.
func (s *State) UnmarshalJSON(data []byte) error {
	type rawTargetWindowState TargetWindowState
	type rawState struct {
		ScheduledWakeEpoch any                             `json:"scheduled_wake_epoch"`
		ExpectingAutoWake  bool                            `json:"expecting_auto_wake"`
		AutoWakeEnabled    *bool                           `json:"auto_wake_enabled,omitempty"`
		LastWakeEpoch      any                             `json:"last_wake_epoch"`
		LastAutoWakeEpoch  any                             `json:"last_auto_wake_epoch"`
		TargetsLastWake    map[string]any                  `json:"targets_last_wake"`
		Targets            map[string]rawTargetWindowState `json:"targets"`
		NotifiedResets     map[string]any                  `json:"notified_resets"`
	}

	var raw rawState
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	s.ScheduledWakeEpoch = parseEpochAny(raw.ScheduledWakeEpoch)
	s.ExpectingAutoWake = raw.ExpectingAutoWake
	s.AutoWakeEnabled = raw.AutoWakeEnabled
	s.LastWakeEpoch = parseEpochAny(raw.LastWakeEpoch)
	s.LastAutoWakeEpoch = parseEpochAny(raw.LastAutoWakeEpoch)

	s.TargetsLastWake = make(map[string]int64, len(raw.TargetsLastWake))
	for k, v := range raw.TargetsLastWake {
		s.TargetsLastWake[k] = parseEpochAny(v)
	}

	s.Targets = make(map[string]TargetWindowState, len(raw.Targets))
	for k, v := range raw.Targets {
		s.Targets[k] = TargetWindowState(v)
	}

	s.NotifiedResets = make(map[string]int64, len(raw.NotifiedResets))
	for k, v := range raw.NotifiedResets {
		s.NotifiedResets[k] = parseEpochAny(v)
	}

	return nil
}

func parseEpochAny(val any) int64 {
	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i
		}
		if f, err := v.Float64(); err == nil {
			return int64(f)
		}
	case string:
		s := strings.TrimSpace(v)
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int64(f)
		}
	}
	return 0
}

func (s *State) normalize() {
	if s.TargetsLastWake == nil {
		s.TargetsLastWake = map[string]int64{}
	}
	if s.Targets == nil {
		s.Targets = map[string]TargetWindowState{}
	}
	if s.NotifiedResets == nil {
		s.NotifiedResets = map[string]int64{}
	}
	// Synchronize legacy TargetsLastWake epoch map with structured Targets map.
	for id, epoch := range s.TargetsLastWake {
		if epoch <= 0 {
			continue
		}
		ts := s.Targets[id]
		if ts.LastWakeAt.IsZero() {
			wakeTime := time.Unix(epoch, 0).UTC()
			ts.LastWakeAt = wakeTime
			if ts.WindowStartedAt.IsZero() {
				ts.WindowStartedAt = wakeTime
			}
			s.Targets[id] = ts
		}
	}
	for id, ts := range s.Targets {
		if !ts.LastWakeAt.IsZero() && s.TargetsLastWake[id] == 0 {
			s.TargetsLastWake[id] = ts.LastWakeAt.Unix()
		}
	}
}

// RecordWakeSuccess records a successful wakeup probe for targetID at wakeTime.
func (s *State) RecordWakeSuccess(targetID string, wakeTime time.Time, remainingPct float64) {
	s.normalize()
	utc := wakeTime.UTC()
	ts := s.Targets[targetID]
	ts.WindowStartedAt = utc
	ts.LastWakeAt = utc
	ts.RemainingPercent = remainingPct
	ts.ConsecutiveFailures = 0
	ts.LastError = ""
	s.Targets[targetID] = ts
	s.TargetsLastWake[targetID] = utc.Unix()
	s.LastWakeEpoch = utc.Unix()
}

// RecordWakeFailure increments the consecutive failure counter for targetID.
func (s *State) RecordWakeFailure(targetID string, errMsg string) {
	s.normalize()
	ts := s.Targets[targetID]
	ts.ConsecutiveFailures++
	ts.LastError = errMsg
	s.Targets[targetID] = ts
}

// LastWakeTime returns the most recent wake timestamp for targetID across structured or legacy fields.
func (s *State) LastWakeTime(targetID string) time.Time {
	if s == nil {
		return time.Time{}
	}
	if ts, ok := s.Targets[targetID]; ok && !ts.LastWakeAt.IsZero() {
		return ts.LastWakeAt
	}
	if epoch, ok := s.TargetsLastWake[targetID]; ok && epoch > 0 {
		return time.Unix(epoch, 0).UTC()
	}
	return time.Time{}
}

// StateStore provides goroutine-safe, atomic persistence for state.json and daemon.log.
type StateStore struct {
	mu       sync.Mutex
	stateDir string
}

// NewStateStore constructs a StateStore rooted at stateDir (defaulting to DefaultStateDir("")).
func NewStateStore(stateDir string) *StateStore {
	if stateDir == "" {
		stateDir = DefaultStateDir("")
	}
	return &StateStore{stateDir: stateDir}
}

// StateDir returns the directory backing state.json and daemon.log.
func (s *StateStore) StateDir() string {
	return s.stateDir
}

// StateFilePath returns the full path to state.json.
func (s *StateStore) StateFilePath() string {
	return filepath.Join(s.stateDir, "state.json")
}

// LogFilePath returns the full path to daemon.log.
func (s *StateStore) LogFilePath() string {
	return filepath.Join(s.stateDir, "daemon.log")
}

// Load reads state.json, returning an initialized default State if absent.
func (s *StateStore) Load() (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	raw, err := os.ReadFile(s.StateFilePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newDefaultState(), nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	st := newDefaultState()
	if err := json.Unmarshal(StripJSONC(raw), st); err != nil {
		return nil, fmt.Errorf("decoding state file: %w", err)
	}
	st.normalize()
	return st, nil
}

// Save atomically persists st to state.json.
func (s *StateStore) Save(st *State) error {
	if st == nil {
		return errors.New("state cannot be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	st.normalize()
	encoded, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}
	encoded = append(encoded, '\n')
	return atomicWriteFile(s.StateFilePath(), encoded, 0o600)
}

// AppendLog writes a structured audit entry to daemon.log, rotating at 5 MB.
func (s *StateStore) AppendLog(level, event, message string, details map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.stateDir, 0o755); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}

	logPath := s.LogFilePath()
	if info, err := os.Stat(logPath); err == nil && info.Size() > maxDaemonLogBytes {
		oldPath := logPath + ".old"
		_ = os.Remove(oldPath)
		_ = os.Rename(logPath, oldPath)
	}

	ts := time.Now().Format("2006-01-02 15:04:05")
	lvl := strings.ToUpper(strings.TrimSpace(level))
	if lvl == "" {
		lvl = "INFO"
	}

	detailSuffix := ""
	if len(details) > 0 {
		if b, err := json.Marshal(details); err == nil {
			detailSuffix = " | " + string(b)
		}
	}

	line := fmt.Sprintf("[%s] [%-5s] [%s] %s%s\n", ts, lvl, event, message, detailSuffix)

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("opening daemon log: %w", err)
	}
	_, writeErr := f.WriteString(line)
	closeErr := f.Close()
	if writeErr != nil {
		return fmt.Errorf("writing daemon log: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing daemon log: %w", closeErr)
	}
	return nil
}

// TailLog returns the last maxLines non-empty lines from daemon.log.
func (s *StateStore) TailLog(maxLines int) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if maxLines <= 0 {
		maxLines = 50
	}

	f, err := os.Open(s.LogFilePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("opening daemon log: %w", err)
	}
	defer f.Close()

	all := []string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := strings.TrimSpace(scanner.Text())
		if txt != "" {
			all = append(all, txt)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning daemon log: %w", err)
	}

	if len(all) <= maxLines {
		return all, nil
	}
	out := make([]string, maxLines)
	copy(out, all[len(all)-maxLines:])
	return out, nil
}
