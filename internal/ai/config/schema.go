package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
)

//go:embed default_config.json
var embeddedDefaultConfig []byte

const SchemaVersion = "1.0.0"

// Config represents the complete aitools configuration schema.
type Config struct {
	Version              string                   `json:"version"`
	DefaultCommitBackend string                   `json:"default_commit_backend"`
	Backends             map[string]BackendConfig `json:"backends"`
	LLM                  LLMConfig                `json:"llm"`
	Commit               CommitConfig             `json:"commit"`
	PR                   PRConfig                 `json:"pr"`
	Daemon               DaemonConfig             `json:"daemon"`
	Notifications        NotificationsConfig      `json:"notifications"`
	Targets              []TargetConfig           `json:"targets"`
	Commands             map[string]CommandConfig `json:"commands"`
}

// BackendConfig defines an AI CLI harness adapter specification.
type BackendConfig struct {
	Name          string              `json:"name,omitempty"`
	Description   string              `json:"description,omitempty"`
	Command       []string            `json:"command"`
	InputMode     string              `json:"input_mode"`
	OutputMode    string              `json:"output_mode"`
	JSONKey       string              `json:"json_key,omitempty"`
	ModelFlag     string              `json:"model_flag,omitempty"`
	EffortFlag    string              `json:"effort_flag,omitempty"`
	DefaultModel  string              `json:"default_model,omitempty"`
	DefaultEffort string              `json:"default_effort,omitempty"`
	CleanANSI     bool                `json:"clean_ansi"`
	CleanMarkdown bool                `json:"clean_markdown"`
	TimeoutSec    int                 `json:"timeout_sec,omitempty"`
	ModeFlags     map[string][]string `json:"mode_flags,omitempty"`
	FormatFlags   map[string][]string `json:"format_flags,omitempty"`
}

// LLMConfig defines default settings for unified non-interactive LLM queries.
type LLMConfig struct {
	DefaultBackend      string `json:"default_backend"`
	DefaultModel        string `json:"default_model,omitempty"`
	DefaultEffort       string `json:"default_effort"`
	DefaultMode         string `json:"default_mode"`
	DefaultOutputFormat string `json:"default_output_format"`
	TimeoutSec          int    `json:"timeout_sec"`
}

// CommitConfig defines conventional commit generation preferences and custom templates.
type CommitConfig struct {
	DefaultBackend         string            `json:"default_backend"`
	Model                  string            `json:"model,omitempty"`
	Effort                 string            `json:"effort"`
	DefaultStageAll        bool              `json:"default_stage_all"`
	MaxDiffLength          int               `json:"max_diff_length"`
	DefaultTemplate        string            `json:"default_template"`
	AutoDetectRepoTemplate bool              `json:"auto_detect_repo_template"`
	TaskID                 string            `json:"task_id,omitempty"`
	PromptTemplate         string            `json:"prompt_template,omitempty"`
	Templates              map[string]string `json:"templates"`
}

// PRConfig defines Pull Request description generation preferences and custom templates.
type PRConfig struct {
	DefaultBackend         string            `json:"default_backend"`
	Model                  string            `json:"model,omitempty"`
	Effort                 string            `json:"effort"`
	DefaultBase            string            `json:"default_base,omitempty"`
	MaxDiffLength          int               `json:"max_diff_length"`
	DefaultTemplate        string            `json:"default_template"`
	AutoDetectRepoTemplate bool              `json:"auto_detect_repo_template"`
	TaskID                 string            `json:"task_id,omitempty"`
	Templates              map[string]string `json:"templates"`
}

// DaemonConfig controls background quota monitoring, RTC wakeups, and notifications.
type DaemonConfig struct {
	AutoWake              AutoWakeConfig    `json:"auto_wake"`
	ActiveHours           ActiveHoursConfig `json:"active_hours"`
	ResetAlertLeadMinutes int               `json:"reset_alert_lead_minutes"`
	DesktopNotifications  bool              `json:"desktop_notifications"`
	Telegram              TelegramConfig    `json:"telegram"`
}

// AutoWakeConfig configures hardware RTC auto-wake and power resuspend policies.
type AutoWakeConfig struct {
	Enabled              bool   `json:"enabled"`
	Time                 string `json:"time"`
	AutoResuspend        bool   `json:"auto_resuspend"`
	ResuspendDelaySec    int    `json:"resuspend_delay_seconds"`
	MaxWakeWindowMinutes int    `json:"max_wake_window_minutes"`
}

// ActiveHoursConfig configures daily active monitoring hours.
type ActiveHoursConfig struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// TelegramConfig holds Telegram Bot webhook credentials.
type TelegramConfig struct {
	Enabled  bool   `json:"enabled"`
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

// NotificationsConfig holds top-level desktop and Telegram notification settings.
type NotificationsConfig struct {
	Desktop  bool           `json:"desktop"`
	Telegram TelegramConfig `json:"telegram"`
}

// TargetConfig defines a monitored AI model or account quota bucket.
type TargetConfig struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Tool                  string   `json:"tool"`
	Account               string   `json:"account"`
	Enabled               bool     `json:"enabled"`
	TelemetryType         string   `json:"telemetry_type"`
	BucketID              string   `json:"bucket_id"`
	WakeCommand           []string `json:"wake_command"`
	StatusCommand         []string `json:"status_command"`
	FallbackIntervalHours int      `json:"fallback_interval_hours"`
	TimeoutSec            int      `json:"timeout_sec"`
}

// CommandConfig defines a declarative custom AI pipeline.
type CommandConfig struct {
	Description    string `json:"description"`
	Backend        string `json:"backend"`
	Model          string `json:"model,omitempty"`
	Effort         string `json:"effort"`
	Mode           string `json:"mode,omitempty"`
	OutputFormat   string `json:"output_format,omitempty"`
	InputType      string `json:"input_type"`
	ShellCommand   string `json:"shell_command,omitempty"`
	PromptTemplate string `json:"prompt_template"`
	MaxInputLength int    `json:"max_input_length,omitempty"`
	TimeoutSec     int    `json:"timeout_sec,omitempty"`
}

const (
	CommitPromptConventional = `You are an expert developer creating a clean conventional git commit message.
Analyze the following git diff and output a single, high-quality conventional commit message.

Requirements:
1. Format: <type>(<scope>): <summary in imperative mood, lower case>
   Types: feat, fix, refactor, docs, chore, test, perf, build, ci
2. Keep the first line under 72 characters.
3. If necessary, add a blank line followed by bullet points describing key architectural details.
4. Output in English.
5. Do NOT include markdown code blocks (no ` + "```" + ` backticks), quotes, or explanation.
6. Do NOT add "Co-Authored-By", AI attribution, or metadata.
7. Output ONLY the raw commit message.

Git Diff:
{diff}`

	CommitPromptTask = `You are an expert developer creating a conventional git commit message linked to a task or issue ticket ({task_id}).
Analyze the following git diff and output a single, high-quality commit message following the organization's task structure.

Context:
- Current Branch: {branch}
- Task / Issue ID: {task_id}
- Modified Files: {files}

Requirements:
1. First line format:
   - If Task ID is present and non-empty: [{task_id}] <type>(<scope>): <summary in imperative mood, lower case>
   - If Task ID is empty or not provided: <type>(<scope>): <summary in imperative mood, lower case>
   Types: feat, fix, refactor, docs, chore, test, perf, build, ci
2. Keep the first line under 72 characters.
3. If necessary, add a blank line followed by concise bullet points describing key changes.
4. If Task ID is present and non-empty, you may include a trailing reference line: "Ref: {task_id}".
5. Output in English.
6. Do NOT include markdown code blocks (no ` + "```" + ` backticks), quotes, or explanation.
7. Do NOT add "Co-Authored-By", AI attribution, or metadata.
8. Output ONLY the raw commit message.

Git Diff:
{diff}`

	CommitPromptTaskScope = `You are an expert developer creating a conventional git commit message where the scope is the task or issue ticket ({task_id}).
Analyze the following git diff and output a single conventional commit message.

Context:
- Current Branch: {branch}
- Task / Issue ID: {task_id}
- Modified Files: {files}

Requirements:
1. First line format:
   - If Task ID is present: <type>({task_id}): <summary in imperative mood, lower case>
   - If Task ID is empty: <type>(<scope>): <summary in imperative mood, lower case>
   Types: feat, fix, refactor, docs, chore, test, perf, build, ci
2. Keep the first line under 72 characters.
3. If necessary, add a blank line followed by bullet points describing key changes.
4. Output in English.
5. Do NOT include markdown code blocks (no ` + "```" + ` backticks), quotes, or explanation.
6. Do NOT add "Co-Authored-By", AI attribution, or metadata.
7. Output ONLY the raw commit message.

Git Diff:
{diff}`

	CommitPromptDetailed = `You are an expert developer creating a comprehensive, well-structured git commit message.
Analyze the following git diff and output a detailed commit message.

Context:
- Current Branch: {branch}
- Task / Issue ID: {task_id}
- Modified Files: {files}

Requirements:
1. Header line: <type>(<scope>): <summary in imperative mood, lower case> (<72 characters)
   Types: feat, fix, refactor, docs, chore, test, perf, build, ci
2. Leave one blank line after the header.
3. Body: A brief 1-2 sentence paragraph explaining WHAT changed and WHY.
4. Followed by bullet points highlighting architectural changes, affected components, or resolved edge cases.
5. If Task ID is present ({task_id}), append a line: "Ticket: {task_id}".
6. Output in English.
7. Do NOT include markdown code blocks (no ` + "```" + ` backticks), quotes, or explanation.
8. Do NOT add "Co-Authored-By", AI attribution, or metadata.
9. Output ONLY the raw commit message.

Git Diff:
{diff}`
)

const (
	PRPromptStandard = `You are an expert software developer and technical lead writing a Pull Request (PR) description in GitHub Flavored Markdown.
Analyze the following branch commits, changed files, and git diff. Generate a comprehensive, professional PR description.

Context:
- Current Branch: {branch}
- Target Base Branch: {base}
- Task / Issue ID: {task_id}
- Commits on Branch:
{commits}
- Modified Files:
{files}
- Diff Summary:
{stat}

Structure the PR description using the following sections:
## Summary & Motivation
Provide a clear, high-level overview of what changes are introduced in this pull request and what problem they solve.
{task_ref}

## Key Changes
- Group bullet points by component or subsystem highlighting the notable features, fixes, or refactors.

## Architecture & Technical Decisions
- Summarize design patterns, trade-offs, or architectural choices applied (omit or keep brief if trivial).

## Verification & Testing
- Document how the changeset was tested (unit test results, linting, manual validation steps, commands executed).

## Checklist
- [x] Code follows repository conventions and architecture patterns
- [x] Relevant tests and quality gates pass cleanly
- [x] Documentation, comments, and configurations updated where appropriate

Requirements:
1. Provide accurate technical summaries based on the real commits and diff.
2. Output in English.
3. Do NOT wrap entire output in a triple backtick code fence.
4. Output ONLY the raw Markdown PR description.

Git Branch Diff:
{diff}`

	PRPromptTask = `You are an expert developer creating a task-linked Pull Request description in GitHub Flavored Markdown.
Analyze the following branch commits, changed files, and git diff to produce a PR description tightly connected to ticket {task_id}.

Context:
- Current Branch: {branch}
- Target Base Branch: {base}
- Task / Issue ID: {task_id}
- Commits:
{commits}
- Modified Files:
{files}

Format:
# [{task_id}] <Descriptive PR Title>

> **Issue / Ticket**: {task_id}

## Problem / Requirement
Explain the user problem, requirement, or bug that prompted this work.

## Proposed Solution & Architecture
Explain how the implementation resolves the ticket cleanly and robustly.

## Detailed Changes
{commits}
Summary of key logic, API, and file modifications.

## Testing & Verification
- Test commands executed and validation proof.

## Checklist
- [x] Resolves ticket requirements
- [x] Tests added or updated
- [x] No breaking changes introduced (or documented below)

Requirements:
1. If Task ID is empty, omit the [{task_id}] prefix in the title.
2. Output in clean GitHub Flavored Markdown.
3. Do NOT wrap the entire response in backticks. Output ONLY the PR description.

Git Branch Diff:
{diff}`

	PRPromptMinimal = `You are an expert developer creating a concise, high-signal Pull Request description.
Analyze the commits and git diff to produce a clean Markdown summary.

Context:
- Branch: {branch} -> {base}
- Task: {task_id}

Format:
## Overview
1-2 sentences summarizing the change and purpose.

## Changes
- Concise bullet points of what was added, modified, or removed.

## Verification
- Note on test suites executed and verification result.

Requirements:
- Punchy, technical, and to the point.
- Output ONLY the raw Markdown content.

Git Branch Diff:
{diff}`

	PRPromptDetailed = `You are a Principal Architect writing a thorough, production-grade Pull Request description.
Analyze the following branch commits and git diff to produce a deep-dive Markdown specification.

Context:
- Branch: {branch} -> {base}
- Task / Issue ID: {task_id}
- Commits:
{commits}
- Changed Files:
{files}
- Diff Stat:
{stat}

Structure:
# <Component / Feature Name>: <Summary of Architecture Change>

## 1. Executive Summary
Clear explanation of the problem, motivation, and scope of this changeset.

## 2. Architecture & Design Patterns
- Detailed breakdown of patterns, libraries, and design decisions applied.
- Edge cases considered and trade-offs made.

## 3. Subsystem Changes
Break down changes by subsystem or folder with rationale for each.

## 4. Breaking Changes & Compatibility
Explicitly state any breaking changes, environment variables, or schema updates (or "No breaking changes").

## 5. Testing & Verification Gates
- Unit test coverage details.
- Integration / verification steps.
- Security and performance considerations.

## 6. Reviewer Guidance
Pointers on where to start reviewing and key files to scrutinize.

Requirements:
- High technical rigor.
- Output ONLY the Markdown PR description.

Git Branch Diff:
{diff}`
)

// BuiltInCommitTemplates returns a fresh copy of the canonical built-in commit prompt templates.
func BuiltInCommitTemplates() map[string]string {
	return map[string]string{
		"conventional": CommitPromptConventional,
		"task":         CommitPromptTask,
		"task-scope":   CommitPromptTaskScope,
		"detailed":     CommitPromptDetailed,
		"jira":         CommitPromptTask,
		"jira-scope":   CommitPromptTaskScope,
	}
}

// BuiltInPRTemplates returns a fresh copy of the canonical built-in PR prompt templates.
func BuiltInPRTemplates() map[string]string {
	return map[string]string{
		"standard": PRPromptStandard,
		"task":     PRPromptTask,
		"minimal":  PRPromptMinimal,
		"detailed": PRPromptDetailed,
	}
}

// EmbeddedDefaultBytes returns a copy of the raw embedded default_config.json bytes.
func EmbeddedDefaultBytes() []byte {
	out := make([]byte, len(embeddedDefaultConfig))
	copy(out, embeddedDefaultConfig)
	return out
}

// DefaultConfig parses and normalizes the embedded default configuration.
func DefaultConfig() (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(embeddedDefaultConfig, &cfg); err != nil {
		return nil, fmt.Errorf("parsing embedded default config: %w", err)
	}
	cfg.normalize()
	return &cfg, nil
}

func (c *Config) normalize() {
	if c.Version == "" {
		c.Version = SchemaVersion
	}
	if c.DefaultCommitBackend == "" {
		c.DefaultCommitBackend = "agy"
	}
	if c.Backends == nil {
		c.Backends = map[string]BackendConfig{}
	}
	if c.Commit.Templates == nil {
		c.Commit.Templates = map[string]string{}
	}
	for k, v := range BuiltInCommitTemplates() {
		if _, exists := c.Commit.Templates[k]; !exists {
			c.Commit.Templates[k] = v
		}
	}
	if c.PR.Templates == nil {
		c.PR.Templates = map[string]string{}
	}
	for k, v := range BuiltInPRTemplates() {
		if _, exists := c.PR.Templates[k]; !exists {
			c.PR.Templates[k] = v
		}
	}
	if c.Targets == nil {
		c.Targets = []TargetConfig{}
	}
	if c.Commands == nil {
		c.Commands = map[string]CommandConfig{}
	}
	if c.Commit.MaxDiffLength <= 0 {
		c.Commit.MaxDiffLength = 15000
	}
	if c.PR.MaxDiffLength <= 0 {
		c.PR.MaxDiffLength = 25000
	}
	if c.LLM.TimeoutSec <= 0 {
		c.LLM.TimeoutSec = 60
	}
	if c.Daemon.AutoWake.Time == "" {
		c.Daemon.AutoWake.Time = "04:00"
	}
	if c.Daemon.AutoWake.ResuspendDelaySec <= 0 {
		c.Daemon.AutoWake.ResuspendDelaySec = 12
	}
	if c.Daemon.AutoWake.MaxWakeWindowMinutes <= 0 {
		c.Daemon.AutoWake.MaxWakeWindowMinutes = 15
	}
	if c.Notifications.Telegram.BotToken != "" && c.Daemon.Telegram.BotToken == "" {
		c.Daemon.Telegram = c.Notifications.Telegram
	}
}

// EffectiveCommitTemplates merges built-in commit templates with user-configured templates.
func (c *Config) EffectiveCommitTemplates() map[string]string {
	merged := BuiltInCommitTemplates()
	maps.Copy(merged, c.Commit.Templates)
	return merged
}

// EffectivePRTemplates merges built-in PR templates with user-configured templates.
func (c *Config) EffectivePRTemplates() map[string]string {
	merged := BuiltInPRTemplates()
	maps.Copy(merged, c.PR.Templates)
	return merged
}
