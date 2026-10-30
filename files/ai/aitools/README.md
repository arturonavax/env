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

### 4. AI-Powered Git Conventional Commit Pipeline (`aitools commit` / `ai-commit` / `ai-cm`)
- **Diff Analysis**: Inspects `git diff --cached` (or automatically prompts to stage modified files if unstaged).
- **Multi-Backend AI Generation**: Queries the selected AI backend (`agy`, `claude`, `codex`, `opencode`, `pi`, or `llm`) with instructions to produce a clean Conventional Commit message without AI fluff or markdown codeblocks. Reuses the unified `aitools llm` engine with `mode: "chat"` and `output_format: "text"`.
- **Interactive Action Loop**:
  - `[a]`: Accept and commit.
  - `[e]`: Edit message in `$EDITOR` before committing.
  - `[r]`: Regenerate with AI.
  - `[c]`: Cancel.

### 5. Unified Non-Interactive LLM CLI Wrapper (`aitools llm` / `ai-llm`)
- **Multi-Harness Encapsulation**: Dispatches prompt queries across any AI CLI tool (`agy`, `claude`, `codex`, `llm`, `opencode`, `pi`, `ollama`, or custom binary) with automatic I/O protocol normalization (`stdin`, `arg`, `temp_file`, `json`).
- **Pipeline Stdin Streaming**: Transparently accepts stdin streams (e.g. `cat file | aitools llm "instruction"`) even if the underlying harness lacks stdin pipeline support.
- **File & Directory Attachments**: Attach files (`-f <path>`) and full directory trees (`-d <path>`) into prompt context, automatically filtering noise directories (`.git`, `node_modules`, `vendor`, etc.) and enforcing size bounds.
- **Execution Modes (`--mode`)**:
  - `chat` / `ask`: Single-turn direct response (read-only planning, no tool/agent loops).
  - `action` / `simple-agent`: Single order-to-action execution.
  - `agent`: Default full agentic capabilities of the harness.
- **Output Formats (`--output-format`)**: Cleaned `text` (default), structured `json` envelope, or unmodified `raw` stdout, with optional `--output-file` destination.

---

## 2. Command Line Interface

```text
Usage: aitools [OPTIONS] COMMAND [ARGS]...

Commands:
  quota         Inspect model quota windows, remaining percentage, and reset times
  watch         Live-updating TUI dashboard for AI quotas and timers
  wake          Evaluate models and trigger wakeup probes for expired/idle quotas
  commit        Generate Conventional Git Commit messages from diff using AI
  llm           Execute non-interactive prompt queries through any AI CLI harness
  backends      List all available AI CLI harnesses and execution strategies
  setup         Launch interactive setup & configuration wizard
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

# Launch interactive configuration wizard (harnesses, models, templates, RTC auto-wake)
aitools setup
aitools config --interactive
# Or launch standalone Pure-Bash wizard:
anx tools ai setup

# --- Unified AI CLI Wrapper (aitools llm) ---
# Direct query using default backend (agy)
aitools llm "Explain Python GIL and concurrency tradeoffs"

# Stream input via pipeline (works even if harness lacks native stdin support)
cat error.log | aitools llm "Diagnose root cause and suggest a fix" -b agy

# Combine piped diff with instructions and model override
git diff | aitools llm "List potential breaking changes" -m gemini-2.5-flash

# Attach files and directory trees into prompt context
aitools llm "Review code structure" -f Makefile -d files/ai/aitools --mode chat

# Restrict execution mode: 'chat' / 'ask' (single-turn), 'action' (single order), 'agent'
aitools llm "Draft a migration plan" --mode chat

# Request structured JSON response envelope
git status | aitools llm "Summarize state" -o json

# Save output directly to file
aitools llm "Generate API client stub" -f schema.json -O client.py

# --- Git AI Conventional Commits (aitools commit / ai-commit / ai-cm) ---

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

# List available commit prompt templates (including discovered repo template)
aitools commit --list-templates

# Commit using task ticket convention (auto-extracted from branch e.g. feature/PROJ-123-auth, fix/#42)
aitools commit --template task

# Commit with manual Task ID and specific model override
aitools commit -t task -k PROJ-999 --model gemini-2.5-flash

# Commit strictly adhering to the repository's own commit template (.gitmessage / commit.template)
aitools commit --template repo

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
| `ai-commit` | `aitools commit` | AI-powered conventional commit from diff |
| `ai-commit-task` | `aitools commit --template task` | AI commit with auto-extracted task/ticket ID |
| `ai-commit-repo` | `aitools commit --template repo` | AI commit strictly adhering to repo's `.gitmessage` |
| `ai-commit-detailed` | `aitools commit --template detailed` | AI commit with multi-line detailed breakdown |
| `ai-cm` | `aitools commit` | Short alias for AI conventional commit |
| `ai-cmk` | `aitools commit --template task` | Short alias for task ticket commit |
| `ai-cmr` | `aitools commit --template repo` | Short alias for repo-templated commit |
| `ai-cmd` | `aitools commit --template detailed` | Short alias for detailed commit breakdown |
| `ai-pr` | `aitools pr` | AI-powered Pull Request Markdown description from branch diff |
| `ai-pr-task` | `aitools pr --template task` | PR description structured with Task / Ticket ID |
| `ai-pr-repo` | `aitools pr --template repo` | PR description strictly following repository PR template |
| `ai-pr-detailed` | `aitools pr --template detailed` | PR description with deep architectural specification |
| `ai-prk` | `aitools pr --template task` | Short alias for task-linked PR description |
| `ai-prr` | `aitools pr --template repo` | Short alias for repository-templated PR description |
| `ai-prd` | `aitools pr --template detailed` | Short alias for detailed PR description |
| `ai-wake` | `aitools wake` | Quota primer probe |
| `ai-quota` | `aitools quota` | Quick quota status inspection |
| `ai-llm` | `aitools llm` | Unified non-interactive AI query wrapper |
| `ai-optimize` | `aitools optimize` | Hardware & security audit and AI optimization script generator |
| `ai-opt` | `aitools optimize` | Short alias for system optimizer |
| `sysopt` | `aitools optimize` | System optimizer alias |

---

## 5. Versatile Commit Templates & Repository Integration

`aitools commit` provides a versatile, fully configurable template engine designed to be completely agnostic to issue trackers (Jira, GitHub Issues, Linear, GitLab, etc.) and seamlessly integrates with repository-level commit standards.

### Built-in Templates

- **`conventional`**: Standard Conventional Commits: `<type>(<scope>): <summary>`
- **`task`**: Task-prefixed format: `[<TASK-ID>] <type>(<scope>): <summary>`
- **`task-scope`**: Task issue as scope: `<type>(<TASK-ID>): <summary>`
- **`detailed`**: Multi-line commit with summary, WHAT/WHY paragraph, bullets, and ticket
- **`repo`**: Repository commit template (automatically discovered from git config or repository files)

### Automatic Repository Template Discovery

When operating inside a Git repository, `aitools` automatically checks for repository-level commit guidelines and templates:
1. `git config commit.template`: Defined locally or globally in git.
2. Well-known template files in the repository root: `.gitmessage`, `.gitmessage.txt`, `.github/commit-template.md`, `.github/commit_template.md`, `.commit-template`.

If detected and `auto_detect_repo_template` is enabled (default: `true`), `aitools` uses the repository's guidelines directly, translating comments and placeholders into an instruction prompt without requiring any manual setup.

### Agnostic Task / Ticket Auto-Detection

If your current Git branch follows ticket naming conventions (e.g. `feature/PROJ-123-auth`, `fix/CORE-942_login`, `issue/42-fix`), `aitools` automatically extracts the task identifier (`PROJ-123`, `CORE-942`, `#42`) and injects it into `{task_id}`. You can override it anytime with `--task <ID>`, `-k <ID>`, or `--ticket <ID>`.

### Template Placeholders

When defining custom templates in `~/.config/aitools/config.json` or external template files, the following placeholders are available:

- `{diff}`: Git staged diff
- `{branch}`: Active Git branch name
- `{task_id}`: Auto-detected or overridden Task/Issue ID (also available via `{issue_id}` or `{ticket}`)
- `{files}`: Comma-separated list of modified files
- `{model}`: Active LLM model name
- `{scope}`: Scope passed via `--scope` / `-s`

---

## 6. AI Pull Request (PR) Description Generator (`aitools pr` / `ai-pr`)

Under the same principles as the commit generator, `aitools pr` analyzes all commits and diffs across a Git branch relative to a target base branch (e.g. `main`, `master`, `develop`) and generates a structured, publication-ready Pull Request Markdown description.

### Key Capabilities
- **Branch Context Extraction**: Gathers commit log (`git log --no-merges base..HEAD`), diff summary (`git diff --stat`), list of modified files, and full diff.
- **Auto-Detection of Base Branch**: Automatically detects target base branch (`origin/main`, `main`, `master`, or upstream remote branch) or accepts explicit `-B <branch>`.
- **Repository PR Template Prioritization**: Automatically discovers and fills out repository PR templates (`.github/pull_request_template.md`, `.github/PULL_REQUEST_TEMPLATE/*.md`, `pull_request_template.md`, `.gitlab/merge_request_templates/`, etc.) with highest priority.
- **Built-in PR Templates**:
  - `standard` (default): Comprehensive PR with Summary, Key Changes, Architecture Decisions, Verification, and Checklist.
  - `task`: Task-linked PR format (`[<TASK-ID>] Title`, Problem, Solution, Verification, Checklist).
  - `minimal`: Terse, high-signal summary with bulleted changes and testing notes.
  - `detailed`: Deep-dive architectural specification with subsystem walkthrough.
- **Interactive & Automation Modes**:
  - Interactive loop: Accept, Copy to clipboard (`wl-copy`/`xclip`/`pbcopy`), Save to file, Edit in `$EDITOR`, Regenerate with AI, or Create PR directly via GitHub CLI (`gh pr create`).
  - Pipeline mode (`--print` / `-p`): Emits raw Markdown directly to stdout for shell pipelines (e.g. `gh pr create --body "$(aitools pr -p)"`).

### Usage Examples

```bash
# Generate PR description for current branch against auto-detected base (main/master)
aitools pr

# Generate PR against specific base branch
aitools pr -B develop

# Generate and copy directly to clipboard
aitools pr -c

# Save PR description directly to Markdown file
aitools pr -o PR_DESCRIPTION.md

# Pipeline directly into GitHub CLI
gh pr create --title "feat: my new feature" --body "$(aitools pr --print)"

# Force specific template
aitools pr --template task
aitools pr --template detailed
aitools pr --template repo

# List available PR templates
aitools pr --list-templates
```

---

## 7. AI Harness Adapters & I/O Protocol Normalization

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

## 8. Multi-Platform AI System Hardware & Performance Optimizer (`anx optimize` / `aitools optimize` / `ai-opt`)

`aitools optimize` audits the host system's hardware, kernel tunables, storage devices, and Lynis security profile, distills high-signal actionable bottlenecks, and delegates to `aitools llm` to generate an ultra-personalized, reversible optimization script backed by strict automated security guardrails.

### Capabilities:
- **Comprehensive Hardware Profiling**: Detects CPU architecture, physical cores, logical threads, scaling governor/driver, total/available RAM, swap size, ZRAM/Zswap status, storage devices (SSD/NVMe vs mechanical HDD), active I/O schedulers, SSD TRIM timer, and GPU/display server.
- **Deep Security & Lynis Audit**: Integrates with Lynis (when installed) for fast, non-intrusive compliance and hardening audits; falls back gracefully to native firewall, kernel pointer exposure, and core dump security checks.
- **Actionable Report Distillation**: Condenses noisy raw outputs into a structured, low-token Markdown digest emphasizing concrete bottlenecks (e.g. NVMe scheduler alignment, swappiness for SSDs, inotify watch expansion for IDEs, BBR congestion control).
- **Automated Security Guardrail Engine**: Inspects generated code line-by-line and strictly blocks execution if any destructive operations are detected (`rm -rf`, `mkfs`, `fdisk`, `dd if=`, raw block device writes, firewall disabling, credential tampering, or remote pipeline executions).
- **Execution Modes**:
  - Interactive Wizard (`--wizard` / `-w`): Step-by-step TUI guided by `wizard_engine.sh`.
  - Dry-Run (`--dry-run` / `-d`, default): Generates, validates guardrails, and previews script without executing.
  - Apply (`--apply` / `-y`): Applies verified optimizations with automatic backups.
  - Audit Only (`--audit-only` / `-a`): Prints distilled hardware & security actionable report without generating code.

### Usage Examples:

```bash
# Print distilled hardware & security actionable report (fast inspection)
aitools optimize --audit-only

# Launch interactive step-by-step optimization wizard
aitools optimize --wizard

# Generate and preview tailored optimization script (dry-run)
aitools optimize

# Bypass Lynis scan for near-instant audit
aitools optimize --skip-lynis

# Save generated script directly to file
aitools optimize -o /tmp/custom_opt.sh

# Apply optimizations with elevated privileges after guardrail verification
aitools optimize --apply

# Validate an external script against the security guardrail engine
aitools optimize --validate ./my_script.sh
```

---

## 9. Configuration Schema (`~/.config/aitools/config.json`)

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
    "auto_detect_repo_template": true,
    "task_id": null,
    "templates": {
      // Custom templates can be registered here:
      // "my-team": "Write commit for {task_id}...\n{diff}"
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
