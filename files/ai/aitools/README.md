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
  backends      List all available AI CLI harnesses and execution strategies
  daemon        Manage background wakeup daemon, RTC alarms, and suspension
  status        Check availability and paths of AI tools and runtimes
  config        Show, edit, or locate configuration file
  run           Execute a custom declarative AI command/pipeline
  commands      List all built-in and declarative custom AI commands
```

### Examples

```sh
# View all available built-in commands and custom declarative pipelines
aitools commands

# Run declarative custom commands (direct invocation or via run)
aitools review                     # AI code review on current staged diff
aitools explain "what is defer in go"  # Explain snippet or concept
aitools ask "difference between slice and array" # Quick one-line question
aitools run optimize "code..."     # Optimize code snippet

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

# List available commit prompt templates
aitools commit --list-templates

# Commit using Jira ticket convention (auto-extracted from branch e.g. feature/PROJ-123-auth)
aitools commit --template jira

# Commit with manual Jira ID and specific model override
aitools commit -t jira -j PROJ-999 --model gemini-2.5-flash

# Multi-line detailed commit with scope override
aitools commit --template detailed --scope auth

# List available AI CLI harness backends and their I/O strategies
aitools backends
aitools commit --list-backends

# Commit using specific AI backend (agy, claude, codex, llm, opencode, pi, ollama)
aitools commit --backend agy
aitools commit --backend llm

# Stage all modified files and generate commit via Claude backend
aitools commit -a --backend claude

# Check status of AI CLIs in PATH
aitools status

# Enable 04:00 AM RTC hardware auto-wake
aitools daemon auto-wake on 04:00
```

---

## 3. Declarative AI Pipelines & Dynamic Commands

`aitools` allows defining custom workflows in `~/.config/aitools/config.json` without recompiling or writing Python code. Any command declared under `"commands"` is automatically exposed as an executable CLI subcommand:

```jsonc
"commands": {
  "review": {
    "description": "Perform an AI code review on current staged git diff",
    "backend": "agy",
    "effort": "low",
    "input_type": "git_staged_diff",
    "prompt_template": "You are a Principal Software Architect. Review the diff:\n\n{input}"
  },
  "explain": {
    "description": "Explain code or error trace",
    "backend": "agy",
    "input_type": "stdin_or_args",
    "prompt_template": "Explain clearly with fundamentals:\n\n{input}"
  }
}
```

Supported `input_type` modes:
- `git_staged_diff`: Automatically feeds `git diff --cached` into `{input}`.
- `git_diff`: Feeds repository working tree diff into `{input}`.
- `stdin_or_args`: Reads from piped stdin (e.g. `cat file | aitools explain`) or command line arguments.
- `args`: Concatenates trailing CLI arguments into `{input}`.
- `stdin`: Reads input directly from standard input until EOF.
- `file`: Reads file content from path specified in first argument.
- `none`: Static prompt execution without input interpolation.

---

## 4. Shell Aliases

Configured in `~/.base.zsh`:

| Alias | Command | Purpose |
| :--- | :--- | :--- |
| `ait` | `aitools` | Fast CLI prefix |
| `gcm` | `aitools commit` | AI-powered conventional commit from diff |
| `gcmj` | `aitools commit --template jira` | AI commit with auto-extracted Jira ticket ID |
| `gcmd` | `aitools commit --template detailed` | AI commit with multi-line detailed breakdown |
| `ai-wake` | `aitools wake` | Quota primer probe |
| `ai-quota` | `aitools quota` | Quick quota status inspection |

---

## 5. Commit Templates & Jira Integration

`aitools commit` supports dynamic templates for organizations requiring specific commit structures (e.g. Jira issue IDs, branch names, scopes).

### Built-in Templates

- **`conventional`**: Standard Conventional Commits: `<type>(<scope>): <summary>`
- **`jira`**: Jira-prefixed format: `[<JIRA-ID>] <type>(<scope>): <summary>`
- **`jira-scope`**: Jira ticket as scope: `<type>(<JIRA-ID>): <summary>`
- **`detailed`**: Multi-line commit with summary, WHAT/WHY paragraph, bullets, and ticket

### Ticket Auto-Detection

If your current Git branch follows ticket naming conventions (e.g. `feature/PROJ-123-auth`, `fix/CORE-942_login`), `aitools` automatically extracts the Jira ID (`PROJ-123`, `CORE-942`) and injects it into `{jira_id}` without requiring manual input. You can override it anytime with `--jira <ID>` or `-j <ID>`.

### Template Placeholders

When defining custom templates in `~/.config/aitools/config.json`, the following placeholders are available:

- `{diff}`: Git staged diff
- `{branch}`: Active Git branch name
- `{jira_id}`: Auto-detected or overridden Jira ticket ID
- `{files}`: Comma-separated list of modified files
- `{model}`: Active LLM model name
- `{scope}`: Scope passed via `--scope` / `-s`

---

## 6. AI Harness Adapters & I/O Protocol Normalization

Different AI CLIs and agent harnesses use incompatible I/O protocols:
- Some tools support **UNIX stdin pipes** (e.g. `llm`, `ollama`), while others require **CLI argument flags** (e.g. `agy -p`, `claude -p`).
- Some harnesses dump terminal colors, spinners, or cursor controls to stdout that corrupt git commit pipelines.
- Specialized harnesses write results to an **output file** (e.g. `--out {output_file}`) or emit **structured JSON** (e.g. `--format json`).

`aitools` implements the **Adapter + Strategy Pattern** (`HarnessAdapter`) to normalize these divergent behaviors:

### Registered Built-in Adapters

| Backend | Input Strategy | Output Strategy | Notes |
| :--- | :--- | :--- | :--- |
| `agy` | `arg` (`-p`) | `stdout` | Strips ANSI escape codes and disables slash command expansion |
| `llm` | `stdin` | `stdout` | Native UNIX pipeline support |
| `claude` | `arg` (`-p`) | `stdout` | Anthropic Claude Code CLI |
| `codex` | `arg` (`exec`) | `stdout` | OpenAI Codex CLI |
| `opencode` | `arg` (`run`) | `stdout` | OpenCode CLI |
| `pi` | `arg` (`ask`) | `stdout` | Inflection Pi CLI |
| `ollama` | `stdin` | `stdout` | Ollama local model runner |

### Custom Harness Registration

You can register custom harnesses in `~/.config/aitools/config.json`:

```jsonc
"backends": {
  "my-file-harness": {
    "command": ["my-agent", "--in", "{prompt_file}", "--out", "{output_file}"],
    "input_mode": "temp_file",
    "output_mode": "temp_file"
  },
  "my-json-harness": {
    "command": ["my-tool", "-p", "{prompt}", "--format", "json"],
    "input_mode": "arg",
    "output_mode": "json",
    "json_key": "choices.0.message.content"
  }
}
```

---

## 7. Configuration Schema (`~/.config/aitools/config.json`)

Supports **JSON with Comments (JSONC)** including `//`, `/* */`, `#`, and trailing commas. Full commented template available at [`files/ai/aitools/config.template.jsonc`](./config.template.jsonc).

```jsonc
{
  "version": "1.0.0",
  "default_commit_backend": "agy",
  "backends": {
    "agy": {
      "command": ["agy", "-p", "{prompt}", "--disable-slash-commands"],
      "input_mode": "arg",
      "output_mode": "stdout"
    },
    "llm": {
      "command": ["llm", "{prompt}"],
      "input_mode": "stdin",
      "output_mode": "stdout"
    }
  },
  "commit": {
    "default_backend": "agy",
    "model": null,
    "effort": "low",
    "default_stage_all": false,
    "max_diff_length": 15000,
    "default_template": "conventional",
    "jira_id": null,
    "templates": {
      // Custom templates can be registered here:
      // "my-team": "Write commit for {jira_id}...\n{diff}"
    }
  },
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
