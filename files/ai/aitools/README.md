# `aitools` — AI Productivity, Quota Management & Automated Workflows Suite

High-performance, multi-command, multi-agent, and multi-account developer CLI for managing and orchestrating local and cloud AI coding assistants.

Built in zero-dependency Python 3 (standard library only) for sub-millisecond execution and complete cross-platform portability across Linux, macOS, WSL, and Android Termux.

---

## 1. Capabilities & Core Subsystems

### 1. Quota & Reset Telemetry (`aitools quota` / `aitools watch`)
- **Real-Time Telemetry**: Queries active model buckets, remaining quota fractions, and reset timestamps directly from CLI telemetry (`agy -p /usage --output-format json`, `claude status`, etc.).
- **Live Terminal Monitor (`aitools watch`)**: Continuous flicker-free TUI dashboard with live countdowns, visual progress bars (`[████████░░] 82%`), and reset timestamps.
- **Configurable Fallback Countdowns**: For blackbox CLIs or offline environments, provides timed fallback countdowns per model/account.

### 2. Intelligent Model Wakeup & Priming (`aitools wake`)
- **Optimal Rolling Window Activation**: Evaluates all configured targets. If an active 5-hour quota window is already running, it skips sending redundant probes to preserve 100% of your tokens.
- **Minimal Wakeup Probes**: Sends 1-token low-effort ping (e.g. `agy -p "Reply with: 1" --effort low`) to expired or idle models to start their rolling refresh window.
- **Multi-Agent / Multi-Tool / Multi-Account**: Out-of-the-box support for:
  - `agy` (Google Antigravity CLI / Gemini 3.8 / Claude / GPT)
  - `claude` (Anthropic Claude Code CLI)
  - `codex` (OpenAI Codex CLI)
  - `opencode` (OpenCode CLI)
  - `pi` (Inflection Pi CLI)
  - `llm` (Datasette LLM CLI)

### 3. Hardware RTC Auto-Wake & Auto-Resuspend (`aitools daemon`)
- **Automated S3 Sleep Cycle**: Automatically schedules hardware RTC alarms (e.g. `04:00 AM`).
- **Wake, Prime & Sleep Sequence**:
  1. System wakes from S3 sleep via ACPI RTC alarm.
  2. Verifies network connectivity (Wi-Fi/Ethernet).
  3. Evaluates all targets and primes expired/idle models.
  4. Schedules next day's 04:00 AM alarm.
  5. If `auto_resuspend` is enabled, returns machine to sleep in 12s to protect battery.
- **Desktop Alerts & Telegram Notifications**: Emits desktop notifications (`notify-send` / `osascript`) and Telegram alerts when quotas are refreshed or before resets.

### 4. AI-Powered Git Conventional Commit Pipeline (`aitools commit` / `gcm`)
- **Diff Analysis**: Inspects `git diff --cached` (or automatically prompts to stage modified files if unstaged).
- **Multi-Backend AI Generation**: Queries the selected AI backend (`agy`, `claude`, `codex`, `opencode`, `pi`, or `llm`) with instructions to produce a clean Conventional Commit message without AI fluff or markdown codeblocks.
- **Interactive Action Loop**:
  - `[a]`: Accept and commit.
  - `[e]`: Edit message in `$EDITOR` before committing.
  - `[r]`: Regenerate with AI.
  - `[c]`: Cancel.

---

## 2. Command Line Interface

```text
Usage: aitools [OPTIONS] COMMAND [ARGS]...

Commands:
  quota         Inspect model quota windows, remaining percentage, and reset times
  watch         Live-updating TUI dashboard for AI quotas and timers
  wake          Evaluate models and trigger wakeup probes for expired/idle quotas
  commit        Generate Conventional Git Commit messages from diff using AI
  daemon        Manage background wakeup daemon, RTC alarms, and suspension
  status        Check availability and paths of AI tools and runtimes
  config        Show, edit, or locate configuration file
```

### Examples

```sh
# View current quota status and reset times
aitools quota

# Continuous live monitor (refreshes every 15s)
aitools watch --interval 15

# Prime expired quotas (skips models whose timers are already running)
aitools wake

# Force wake probes across all models regardless of active status
aitools wake --force

# Generate Conventional Commit from staged changes
aitools commit

# Stage all modified files and generate commit via Claude backend
aitools commit -a --backend claude

# Check status of AI CLIs in PATH
aitools status

# Enable 04:00 AM RTC hardware auto-wake
aitools daemon auto-wake on 04:00
```

---

## 3. Shell Aliases

Configured in `~/.base.zsh`:

| Alias | Command | Purpose |
| :--- | :--- | :--- |
| `ait` | `aitools` | Fast CLI prefix |
| `gcm` | `aitools commit` | AI-powered conventional commit from diff |
| `ai-wake` | `aitools wake` | Quota primer probe |
| `ai-quota` | `aitools quota` | Quick quota status inspection |

---

## 4. Configuration Schema (`~/.config/aitools/config.json`)

```json
{
  "version": "1.0.0",
  "default_commit_backend": "agy",
  "daemon": {
    "auto_wake": {
      "enabled": false,
      "time": "04:00",
      "auto_resuspend": true,
      "resuspend_delay_seconds": 12,
      "max_wake_window_minutes": 15
    },
    "active_hours": {
      "start": "03:55",
      "end": "22:00"
    },
    "reset_alert_lead_minutes": 5,
    "desktop_notifications": true,
    "telegram": {
      "enabled": false,
      "bot_token": "",
      "chat_id": ""
    }
  },
  "targets": [
    {
      "id": "gemini",
      "name": "Google Gemini (Antigravity)",
      "tool": "agy",
      "account": "default",
      "enabled": true,
      "telemetry_type": "agy_usage",
      "bucket_id": "gemini-5h",
      "wake_command": ["agy", "-p", "Reply with: 1", "--model", "gemini-3.8-flash-low", "--effort", "low"],
      "status_command": ["agy", "-p", "/usage", "--output-format", "json"],
      "fallback_interval_hours": 5,
      "timeout_sec": 20
    }
  ]
}
```
