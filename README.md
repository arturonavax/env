# 🦊 arturonavax/env

High-performance, modular dotfiles repository and multi-platform development environment.

Designed for software engineers, architects, and AI developers who demand near-instant startup times, rock-solid stability, and ergonomic command-line workflows.

---

## Table of Contents

1. [Architectural Principles & Philosophy](#1-architectural-principles--philosophy)
2. [Quick Start & Installation](#2-quick-start--installation)
   - [Interactive Setup Wizard (Upfront Interview)](#interactive-setup-wizard-upfront-interview)
   - [Automated Non-Interactive Installation (Headless / CI-CD)](#automated-non-interactive-installation-headless--ci-cd)
   - [Dry-Run Planning & Manifest Preview](#dry-run-planning--manifest-preview)
3. [Subsystems & Tooling Stack](#3-subsystems--tooling-stack)
   - [1. Base Languages & Toolchains (`requirements`)](#1-base-languages--toolchains-requirements)
   - [2. Modern GPU Terminal & Lean Shell (`terminal`)](#2-modern-gpu-terminal--lean-shell-terminal)
   - [3. Code Editor Suite (`editor` — Neovim & LazyVim)](#3-code-editor-suite-editor--neovim--lazyvim)
   - [4. AI Productivity & Automation Suite (`ai` — `aitools`)](#4-ai-productivity--automation-suite-ai--aitools)
   - [5. OS Desktop Integrations & Window Switcher (`osconfig` — `rcmd`)](#5-os-desktop-integrations--window-switcher-osconfig--rcmd)
   - [6. Patched Nerd Fonts (`fonts`)](#6-patched-nerd-fonts-fonts)
4. [Modern CLI Tool Replacements & Non-Destructive Aliases](#4-modern-cli-tool-replacements--non-destructive-aliases)
5. [Safeguards, Backups & Transactional Rollback](#5-safeguards-backups--transactional-rollback)
6. [How to Customize to Your Liking](#6-how-to-customize-to-your-liking)
7. [Synchronization & Day-to-Day Lifecycle](#7-synchronization--day-to-day-lifecycle)
8. [Quality Assurance & Automated Testing](#8-quality-assurance--automated-testing)
9. [Supported Platforms](#9-supported-platforms)

---

## 1. Architectural Principles & Philosophy

- **Zero-Friction Performance**: Sub-millisecond terminal startup. Startup scripts (`.zshrc`, `.base.zsh`, `.tools.sh`) are compiled into `.zwc` word-code binaries via `zcompile` and memory-mapped by Zsh. Completion caches (`compinit -C`) bypass costly disk audits if validated within 24 hours.
- **Non-Destructive Alias Architecture**: Standard Unix utilities (`grep`, `find`, `cat`, `less`) **are NEVER hijacked or aliased** to modern tools (`rg`, `fd`, `bat`). POSIX purity and muscle memory for remote SSH production environments are strictly preserved, while dedicated shortcuts (`rgh`, `fdh`, `preview`, `dft`, `l`) provide ergonomic speed.
- **Smart Preservation Policy (`smart-append-first`)**: Your existing shell configurations (`~/.zshrc`, `~/.bashrc`) and window bindings are never overwritten. Repository loaders are isolated within `# >>> arturonavax/env >>>` marker blocks, preserving 100% of your personal aliases, history, and secrets.
- **Transactional Execution & LIFO Rollback Engine**: Operates under an audit journal (`src/lib/transaction.sh`). On keyboard interrupt (`Ctrl+C`) or unexpected stage failure, changes are reversed in reverse-chronological order (LIFO), cleanly removing symlinks/files and restoring initial snapshots.
- **Zero Compilation Overhead**: Strictly avoids slow compilation or bloated package managers. Official standalone pre-compiled release binaries (musl-static Rust, Go) are prioritized for near-instant provisioning.
- **Offline & Private**: Embedded SQLite shell history (`atuin`) with directory scoping and zero remote sync or telemetry.

---

## 2. Quick Start & Installation

### Interactive Setup Wizard (Upfront Interview)

Run the guided setup wizard directly from the remote subdomain:

```sh
bash <(curl -fsSL "https://env.arturonavax.dev/install.sh") --wizard
```

Or clone the repository and run locally:

```sh
git clone https://github.com/arturonavax/env.git ~/repos/github.com/arturonavax/env
cd ~/repos/github.com/arturonavax/env
./install.sh --wizard
```

The wizard conducts an **upfront parameter interview**—asking for all destination paths, subsystem components, runtime strategies (GVM vs standalone Go, pnpm vs npm), and OS integrations—before starting autonomous execution. It operates under a transactional audit journal with graceful traps (Ctrl-C / errors) and LIFO rollback protection.

```text
╭────────────────────────────────────────────────────────────────────╮
│  arturonavax/env · Master Environment Wizard                      │
├────────────────────────────────────────────────────────────────────┤
│  High-performance modular development environment.                 │
│  Total Stages: 6 stages planned                                    │
│  Stop anytime with Ctrl+C. Safe idempotent state recovery.         │
╰────────────────────────────────────────────────────────────────────╯
```

---

### Automated Non-Interactive Installation (Headless / CI-CD)

For unattended provisioning, scripts, or container builds, use the non-interactive mode:

```sh
# Complete autonomous installation (all subsystems, zero prompts)
./install.sh -y

# Select specific subsystems with automatic rollback on error
./install.sh -y --components=terminal,editor,ai --rollback-on-error

# Direct subsystem arguments
./install.sh terminal editor ai
```

#### CLI Flags & Options

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-w`, `--wizard` | Launch interactive upfront interview wizard | TTY default |
| `-y`, `--yes`, `--non-interactive` | Run fully unattended non-interactive installation | Headless default |
| `-c`, `--components=<list>` | Comma-separated list: `requirements`, `terminal`, `editor`, `ai`, `fonts`, `osconfig`, `all` | `all` |
| `--dest-bin=<path>` | Destination path for user binaries | `~/.local/bin` |
| `--backup-dir=<path>` | Destination path for configuration backup snapshots | `~/.arturonavax-env-backups` |
| `--skip-backup` | Bypass initial configuration snapshot creation | Disabled |
| `--go-manager=<strategy>` | Go manager strategy: `gvm` (recommended), `standalone`, `package` | `gvm` |
| `--node-manager=<manager>` | Node.js package manager: `pnpm` (recommended), `npm` | `pnpm` |
| `--rollback-on-error` | Automatically reverse all session mutations if any stage fails | Disabled |
| `-p`, `--plan`, `--dry-run` | Safe preview of translated packages, binaries, and touched files | - |
| `--uninstall-plan` | Safe preview of purged files, binaries, desktop files, and caches | - |
| `--check` | Run platform discovery and hardware diagnostic check | - |
| `-h`, `--help` | Display usage documentation and flags | - |

---

### Dry-Run Planning & Manifest Preview

Inspect the full filesystem interaction manifest before making any changes:

```sh
# Preview complete installation manifest and touched files
./install.sh --plan
# Or via make:
make plan

# Preview scoped subsystem plan (e.g. terminal only)
./install.sh plan terminal

# Preview uninstallation teardown manifest (purged binaries, desktop files, caches)
./uninstall.sh --plan
# Or via make:
make uninstall-plan
```

---

## 3. Subsystems & Tooling Stack

### 1. Base Languages & Toolchains (`requirements`)
- **Compilers**: Native build-essential, GCC/Clang, Make, pkg-config, SSL/SQLite development headers.
- **Go (Golang)**: Multi-tier toolchain featuring **GVM (Go Version Manager)** as primary manager, backed by standalone JSON release installer ([`src/remotes/install_golang.sh`](./src/remotes/install_golang.sh)) and native package manager fallbacks.
- **Node.js**: Installed via Fast Node Manager (`fnm`), configured with `pnpm` as the primary disk-efficient package manager. Corepack is decoupled to prevent package manager interference.
- **Rust**: Official `rustup` toolchain with `cargo`.
- **Python**: Modular Python environment with `pyenv` and `pipx`.

### 2. Modern GPU Terminal & Lean Shell (`terminal`)
- **Terminal Emulator**: [Ghostty](https://ghostty.org/) (GPU-accelerated, native Zig implementation).
- **Agent Multiplexer**: [Herdr](https://herdr.dev/) (Native agent state detection, workspace management, zero nested-PTY scroll friction; completely replaces Tmux).
- **Shell**: Lean Zsh with `.zwc` word-code compilation and memory-mapped fast loading (`~/.base.zsh` and `~/.tools.sh`).
- **Prompt**: [Starship](https://starship.rs/) configured with a compact 2-line layout, sub-millisecond execution, and AI agent workspace detection (`🤖 AGENTS`).
- **Shell History**: [Atuin](https://atuin.sh/) SQLite history engine with directory scoping (`filter_mode = "directory"`), durations, exit codes, and zero telemetry (`auto_sync = false`).
- **Tab Completion**: `fzf-tab` interactive search with live contextual previews via `bat` (files), `eza` (directories), `git log` (branches), and `ps` (processes).

### 3. Code Editor Suite (`editor` — Neovim & LazyVim)
- **Engine**: Neovim v0.12+ built on [LazyVim](https://www.lazyvim.org/).
- **Exclusions Architecture** ([`files/nvim/lua/config/exclusions.lua`](./files/nvim/lua/config/exclusions.lua)): Centralized single source of truth ignoring `node_modules`, `vendor`, `.git`, `.venv`, `target`, `dist`, `build`, etc. In-memory O(1) hash table with a 1,000-entry memoization cache for sub-millisecond LSP/completion performance.
- **LSP & Tooling**: Integrated Mason installer, Treesitter AST parsers, `nvim-lint`, and `conform.nvim` formatters.
- **Git & Productivity**: `lazygit`, `grug-far.nvim` AST project search, `snacks.nvim`, and AI assistants (`Avante`, `CodeCompanion`).

### 4. AI Productivity & Automation Suite (`ai` — `aitools`)
- **`aitools` CLI** ([`files/ai/aitools/`](./files/ai/aitools/)):
  - **Live Quota Telemetry (`aitools quota` / `aitools watch`)**: Inspects rolling quota windows, percentages, and reset countdowns across Google Gemini (`gemini-5h`), Claude/GPT (`3p-5h`), Codex, OpenCode, Pi, and LLM.
  - **Selective Idempotent Wakeup (`aitools wake`)**: Automatically skips sending wake probes if the rolling 5-hour window is already ticking, preserving 100% of unused tokens. Sends low-effort 1-token probes only when expired.
  - **Hardware RTC Auto-Wake & Auto-Resuspend (`aitools daemon`)**: Wakes the machine at 04:00 AM via motherboard RTC alarm (`rtcwake`), verifies network, primes expired models, schedules next day's alarm, and suspends back to sleep in 12s. Emits desktop alerts and Telegram notifications.
  - **AI Conventional Commit Pipeline (`aitools commit` / `gcm`)**: Inspects staged diffs and generates Conventional Commit messages via LLM (`agy`, `claude`, `codex`, `opencode`, `pi`, `llm`) with interactive accept/edit/regenerate.
  - **Declarative AI Pipelines (`aitools <cmd>` / `aitools run <cmd>`)**: Extensible zero-code commands declared in `~/.config/aitools/config.json` (`review`, `explain`, `ask`, `optimize`). Rich JSONC format with inline comments and trailing commas. Inspect all pipelines via `aitools commands`.
- **RTK (Rust Token Killer)**: High-speed proxy compressing LLM command outputs by up to 90%.
- **MCP Ecosystem**: Configured Model Context Protocol servers and prompt harnesses.

### 5. OS Desktop Integrations & Window Switcher (`osconfig` — `rcmd`)
- **`rcmd` Window Switcher** ([`files/rcmd/`](./files/rcmd/)): Instant application switcher using **Right-Alt** key combinations (e.g. `Right-Alt + C` focuses Google Chrome).
  - Agnostic multi-backend router: GNOME Shell D-Bus extension ([`rcmd-shell@arturonavax.dev`](./files/gnome/extensions/rcmd-shell@arturonavax.dev/)), Hyprland, Sway, KDE KWin, and X11 fallback (`wmctrl`).
- **Linux Core Tuning**: Increased file descriptor limits (`limits.conf` to 65,536), inotify watchers, and GTK 4.14 renderer stability (`GSK_RENDERER=gl`).
- **XFCE Pro Workflow** ([`src/osconfig/xfce/`](./src/osconfig/xfce/)): Instant screen capture, OCR clipboard extraction (`Super + Shift + C`), and hardened session unlock guard.
- **macOS Developer Tweaks** ([`src/osconfig/macos/`](./src/osconfig/macos/)): Touch ID for terminal `sudo`, Mission Control animation speedup, and Safari Web Inspector tweaks.

### 6. Patched Nerd Fonts (`fonts`)
- Installs patched mono fonts: **MonaspiceNe Nerd Font**, **Cascadia Code**, and **JetBrains Mono**.

---

## 4. Modern CLI Tool Replacements & Non-Destructive Aliases

Standard Unix utilities (`grep`, `find`, `cat`, `less`) **remain completely untouched POSIX commands** to protect scripts and preserve muscle memory when SSHing into remote servers. Ergonomic shortcuts provide modern speed:

| Domain | Classic Tool | Modern Replacement | Rationale & Capabilities | Non-Colliding Shortcut |
| :--- | :--- | :--- | :--- | :--- |
| **AST Diffing** | `diff` / `wdiff` | **`difftastic` (`difft`)** | Tree-sitter structural diffing; compares syntax trees rather than lines. | `dft` |
| **Git Pager** | `diff` / `less` | **`git-delta` (`delta`)** | Syntax-highlighting side-by-side terminal diff viewer. | Default Git pager |
| **Universal SQL** | `litecli` / `pgcli` | **`usql`** | Single Go static binary for Postgres, SQLite, MySQL, Redis, ClickHouse. | `usql` |
| **Instant Help** | `tldr` (npm/python) | **`tealdeer` (`tldr`)** | Native Rust tldr client with instant sub-millisecond responses. | `tldr` |
| **JSON Inspection** | `jq` | **`fx`** | Interactive dual TUI and JavaScript/JSON streaming viewer. | `fx` |
| **Token Killer** | Terminal output | **`rtk`** | Compresses command outputs by up to 90% before sending to LLM context. | `rtk` |
| **Shell History** | `~/.zsh_history` | **`atuin`** | Embedded SQLite history engine with directory scoping; offline and private. | `Ctrl+R` |
| **AI Commits** | Manual commit | **`aitools commit`** | AI Conventional Commit generation from git diff. | `gcm` |
| **File Listing** | `ls` | **`eza`** | Modern file listing with icons, git status, and tree view. | `l`, `ll`, `la`, `lt` |
| **File Search** | `find` | **`fd`** | Fast directory traversal ignoring `.git`. | `fdh` (`fd --hidden`) |
| **Code Search** | `grep` | **`ripgrep` (`rg`)** | Ultra-fast recursive regex search ignoring `.git`. | `rgh` (`rg --hidden`) |
| **File Preview** | `cat` / `less` | **`bat`** | Syntax-highlighted file pager with line numbers. | `preview` |

---

## 5. Safeguards, Backups & Transactional Rollback

### Automatic Snapshot Creation & Quota Rotation
Before modifying any configuration files, the installer automatically creates a timestamped snapshot in:
`~/.arturonavax-env-backups/YYYY-MM-DD_HH-MM-SS/`

- **50MB Quota Rotation**: Automatically purges the oldest snapshots when the total backup size exceeds 50MB using POSIX-portable commands (`du -k`, `sort`).
- **Instant Rollback**:
  ```sh
  # List all available snapshots
  ./utils/restore_backup.sh -l

  # Restore the most recent snapshot
  ./utils/restore_backup.sh -latest

  # Restore a specific timestamp
  ./utils/restore_backup.sh 2026-10-28_12-00-00
  ```

### Transactional Audit Journal & Graceful Shutdown (`src/lib/transaction.sh`)
- Tracks file creation, symlinks, directories, and marker blocks in an audit journal (`~/.arturonavax-env-backups/transactions/`).
- Arms signal traps for `SIGINT` (Ctrl+C) and `SIGTERM`.
- Reverses mutations in **LIFO order** (reverse chronological), cleanly deleting newly created files and restoring original configurations.

---

## 6. How to Customize to Your Liking

This repository is designed to be easily tailored to your personal workflow:

### Custom Shell Aliases & Functions
Add your personal aliases and functions to `~/.zshrc` (anywhere outside the `# >>> arturonavax/env >>>` marker blocks) or create a file at `~/.alias` or `~/.zsh_alias`. They will be preserved across all installations and updates:

```sh
# Example ~/.zshrc additions:
alias myproject="cd ~/repos/myproject && herdr"
export OPENAI_API_KEY="sk-..."
```

### Custom Neovim Plugins (LazyVim)
Add any custom Lua plugin specifications to `~/.config/nvim/lua/plugins/` (e.g. `~/.config/nvim/lua/plugins/my_plugins.lua`):

```lua
return {
  { "tpope/vim-surround" },
  {
    "folke/trouble.nvim",
    cmd = { "TroubleToggle", "Trouble" },
    opts = { use_diagnostic_signs = true },
  },
}
```

### Custom `rcmd` Window Switcher Shortcuts
Edit `~/.config/rcmd/rcmd.conf` to bind your favorite applications to **Right-Alt + Key**:

```ini
c=google-chrome-stable class launch:google-chrome-stable
e=com.mitchellh.ghostty class launch:ghostty
v=cursor class launch:cursor
w=WhatsApp title launch:google-chrome-stable --app=https://web.whatsapp.com
```

### Custom AI Models & Quota Targets
Edit `~/.config/aitools/config.json` to register new models, accounts, or CLI assistants:

```json
{
  "targets": [
    {
      "id": "my-custom-model",
      "name": "My Custom AI Model",
      "tool": "claude",
      "account": "work",
      "enabled": true,
      "telemetry_type": "timed_fallback",
      "wake_command": ["claude", "-p", "Reply with: 1"],
      "fallback_interval_hours": 5,
      "timeout_sec": 25
    }
  ]
}
```

---

## 7. Synchronization & Day-to-Day Lifecycle

To propagate updates from the repository into your active environment without running a full reinstallation:

```sh
# Synchronize all components
./utils/sync_config.sh all

# Or synchronize specific targets:
./utils/sync_config.sh terminal     # Ghostty, Herdr, Zsh, Starship, Atuin
./utils/sync_config.sh editor       # Neovim configs & Lua exclusions
./utils/sync_config.sh plugins      # Restore LazyVim plugins from lockfile
./utils/sync_config.sh plugins-update # Update LazyVim plugins and update repo lockfile
./utils/sync_config.sh devtools     # Formatters, linters, RTK, and aitools
./utils/sync_config.sh osconfig     # rcmd window switcher and desktop settings
./utils/sync_config.sh vscode       # VSCode settings & keybindings
./utils/sync_config.sh cursor       # Cursor IDE settings & keybindings
```

### Clean Teardown & Uninstallation
To cleanly purge packages, binaries, systemd units, and local configuration files:

```sh
# Preview teardown manifest without touching anything
./uninstall.sh --plan

# Run clean uninstallation
./uninstall.sh
```

---

## 8. Quality Assurance & Automated Testing

The repository includes a comprehensive automated test harness ensuring zero regression errors:

```sh
# Run the complete test harness (syntax, static analysis, unit, integration)
make test
# Or directly: ./tests/run_tests.sh

# Run individual test suites:
make lint              # Syntax validation (bash -n) across 100+ scripts & ShellCheck across 36+ scripts
make test-unit         # 12 Unit suites (platform, packages, smart_append, backup, concurrency, transaction, aitools, shell_configs, etc.)
make test-integration  # Integration smoke test suite
make check             # System hardware and platform pre-flight check
```

---

## 9. Supported Platforms

The universal package matrix ([`src/lib/packages.sh`](./src/lib/packages.sh)) translates package names natively across:

- **Debian / Ubuntu / Linux Mint / Pop!_OS** (`apt`)
- **Fedora / RHEL / CentOS / Rocky Linux / AlmaLinux** (`dnf`)
- **Arch Linux / EndeavourOS / Manjaro / Garuda** (`pacman`)
- **openSUSE / SLES** (`zypper`)
- **Alpine Linux** (`apk`)
- **macOS** (Apple Silicon arm64 & Intel x86_64 via `brew`)
- **Android (Termux)** (`pkg`, single-user sandbox without root)
- **Windows** (WSL 1/2, MSYS2, Git Bash)

---

## 10. Remote Scripts Subdomain

Autonomous self-contained scripts can be executed directly via `curl`:

- `env.arturonavax.dev`
- `env.arturonavax.com`

Examples:

```sh
# Guided master setup wizard
bash <(curl -fsSL "https://env.arturonavax.dev/install.sh") --wizard

# Standalone Go runtime installer (API-driven)
bash <(curl -fsSL "https://env.arturonavax.dev/install_golang.sh") -i

# Hardware LED Scroll Lock activation
bash <(curl -fsSL "https://env.arturonavax.dev/activate_scroll_lock.sh")
```
