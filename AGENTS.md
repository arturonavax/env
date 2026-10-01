# AI Agent Engineering Guide — env

Comprehensive operational guide and architecture reference for AI coding agents and engineers working in `arturonavax/env`.

---

## 1. System Overview & Entry Points

`arturonavax/env` is a high-performance, modular dotfiles repository and multi-platform development environment supporting Linux (Debian/Ubuntu, Arch, Fedora), macOS, and Windows (PowerShell/Windows Terminal).

### Repository Structure

```text
├── install.sh              # Primary system installation entry point
├── uninstall.sh            # Systematic uninstallation and cache purging entry point
├── files/                  # Canonical configuration files deployed to target systems
│   ├── nvim/               # Neovim (LazyVim) modular configuration and exclusions
│   ├── rcmd/               # Window switcher binary, configuration, and backend scripts
│   ├── gnome/extensions/   # GNOME Shell extensions (rcmd-shell@arturonavax.dev)
│   ├── ghostty/            # Ghostty terminal configurations (launches herdr)
│   ├── herdr/              # Herdr agent multiplexer configuration (config.toml)
│   ├── atuin/              # Atuin SQLite shell history configuration (config.toml)
│   ├── ai/                 # AI prompt templates, MCP harnesses, and RTK configs
│   ├── zsh/                # Zsh config, plugins, tools, and color definitions
│   ├── starship/           # Cross-shell Starship prompt configuration
│   └── [cursor,vscode,...] # IDE configs, settings, and formatters
├── src/                    # Subsystem installation scripts
│   ├── install_terminal.sh # Ghostty, Herdr, zsh, fonts setup
│   ├── install_editor.sh   # Neovim, Mason dependencies, LazyVim sync
│   ├── install_ai.sh       # CLI AI assistants, RTK optimizer, MCP harnesses, prompt configs
│   ├── _install_requirements.sh # Core system packages & package managers
│   ├── requirements/       # Pre-flight requirement checks (editor, terminal, ai)
│   ├── remotes/            # Self-contained remote scripts executable via curl
│   └── dom-scraping/       # Browser/node utility scraping scripts
├── utils/                  # Safe system lifecycle & synchronization utilities
│   ├── sync_config.sh      # Propagate repo dotfiles into the active filesystem
│   ├── backup_config.sh    # Snapshot active user configurations to ~/.arturonavax-env-backups/
│   ├── copy_config.sh      # Ingest active user dotfiles back into the repo
│   └── diff_snapshots.sh   # Review differences between local machine and repo
├── .atl/                   # Agent runtime state (skill registry index & cache)
├── codegraph.json          # CodeGraph AST indexing configuration and exclusion globs
└── .engram/config.json     # Engram memory project binding ("env")
```

### Lifecycle Entry Points

- **`./install.sh [args]`**:
  - Automatically triggers `./utils/backup_config.sh` before mutating files.
  - Parameters: `requirements` (`r`), `fonts` (`f`), `terminal` (`t`), `editor` (`e`), `ai`, `osconfig` (`o`), `all` (`a`).
  - Acquires sudo credentials upfront and runs a background keep-alive loop (`sleep 60; kill -0 "$$"`).
- **`./uninstall.sh`**:
  - Two-phase teardown:
    1. Uninstalls packages (apt, snap, brew), systemd user units, desktop entries, and `/opt` / `/usr/local/bin` binaries.
    2. Purges configuration directories (`~/.config/nvim`, `~/.config/ghostty`, `~/.config/herdr`), caches (`~/.cache/nvim`), and local states (`~/.local/state/nvim`, `~/.local/share/nvim`).
- **`./utils/sync_config.sh [target]`**:
  - Selectively applies configuration updates without a complete installation cycle. Targets: `terminal`, `editor`, `plugins`, `plugins-update`, `osconfig`, `vscode`, `cursor`, `devtools`, `all`.
- **`./utils/backup_config.sh`**:
  - Creates timestamped snapshots in `~/.arturonavax-env-backups/YYYY-MM-DD_HH-MM` before destructive operations (capped at 50MB rotation).

---

## 2. Core Architecture Subsystems

### Window Switcher (`rcmd`)

`files/rcmd/` provides a macOS `rcmd`-style instant application launcher and window switcher for Linux:

- **Hotkeys**: Dispatches via Right-Alt key combinations (e.g. `Right-Alt + C` focuses Google Chrome).
- **Agnostic Multi-Backend Router (`files/rcmd/rcmd`)**:
  - **GNOME Shell (Wayland/X11)**: Communicates directly via D-Bus (`busctl`/`gdbus`) with the bundled extension `rcmd-shell@arturonavax.dev` (`files/gnome/extensions/rcmd-shell@arturonavax.dev/extension.js`) exporting interface `dev.arturonavax.rcmd`. Falls back to `org.gnome.Shell.Extensions.RunOrRaise`.
  - **Hyprland**: Dispatches via `hyprctl dispatch focuswindow` and `hyprctl clients -j`.
  - **Sway**: Dispatches via `swaymsg [app_id=...] focus` and IPC trees.
  - **KDE KWin**: Dispatches via KWin scripting over D-Bus (`org.kde.KWin`).
  - **X11 Fallback**: Universal EWMH control via `wmctrl` and `xdotool`.
- **Matching Modes**: Supports matching by `class` (WM_CLASS/app_id), `title` (window title regex for web apps like WhatsApp Web), and dynamic letter fallback.

### Neovim & Exclusions Architecture (LazyVim)

`files/nvim/` is built on top of [LazyVim](https://www.lazyvim.org/) and heavily modularized:

- **Plugins (`files/nvim/lua/plugins/`)**: Specialized specs for `lsp.lua`, `coding.lua`, `snacks.lua`, `ai.lua` (Avante, CodeCompanion), `database.lua`, `gitsigns.lua`, `linting.lua`, `symbol-usage.lua`, `ui-perf.lua`, `web3.lua`.
- **Exclusions Architecture (`files/nvim/lua/config/exclusions.lua`)**:
  - Centralized single source of truth for ignoring noisy, generated, or heavy directories across all editor plugins.
  - **Directories (`M.dirs`)**: `node_modules`, `.pnpm-store`, `vendor`, `.git`, `.venv`, `target`, `dist`, `build`, `artifacts`, `cache`, `coverage`, `.idea`, `.vscode`, etc.
  - **Files (`M.files`)**: Lockfiles (`package-lock.json`, `Cargo.lock`, `poetry.lock`), minified files (`*.min.js`), and sourcemaps (`*.map`).
  - **Exposed Configurations**:
    - `M.picker_exclude`: Injected into Snacks Picker (`files`, `grep`, `smart`).
    - `M.wildignore`: Configured in Neovim options.
    - `M.rg_extra_args`: Ripgrep flags passed to `grug-far.nvim` (`-g !node_modules -g !...`).
    - `M.gopls_directory_filters`: Passed to `gopls` language server settings.
    - `M.lsp_exclude_dirs`: Directory globs for LSP watchers (`vtsls`, `pyright`).
    - `M.is_path_excluded(path)` / `M.is_excluded_name(name)`: In-memory O(1) hash table lookup with memoization cache (1,000 entries) for sub-millisecond evaluation in completion filters.

### Agent Multiplexer (`herdr`)

`herdr` ([herdr.dev](https://herdr.dev/)) is the exclusive terminal and AI agent multiplexer in this environment, replacing Tmux as the default workspace manager launched by Ghostty (`files/ghostty/config` -> `command = zsh -ic "herdr"`):

- **Architecture**: Statically-linked single binary installed to `~/.local/bin/herdr` (and symlinked to `/usr/local/bin/herdr`).
- **Core Capabilities**:
  - Native agent state detection (`working` vs `blocked` spinners/toasts).
  - Multi-workspace supervision without complex pane scripting.
  - Local Unix socket API (`herdr workspace`, `herdr pane`, `herdr wait`) for programmatic agent orchestration.
  - Zero nested-PTY scroll friction and robust terminal protocol parsing.
- **Tmux Elimination**: Tmux compatibility has been completely purged across all dotfiles, VSCode/Cursor configs, and install scripts in favor of Herdr.

### Lean Zsh, FZF & Starship Architecture

To maximize terminal responsiveness and eliminate subshell latency, shell initialization enforces ultra-lean patterns:

- **Cached Completion Initialization**:
  - `compinit -C` bypasses expensive security audit scans if the completion dump (`~/.zcompdump`) is less than 24 hours old.
  - Startup files (`~/.zshrc`, `~/.base.zsh`, `~/.tools.sh`, `~/.lscolors.sh`, and `~/.zcompdump`) are compiled to word-code (`.zwc`) binaries via `zcompile` and memory-mapped (`mmap`) by Zsh, achieving sub-millisecond parsing.
  - Includes a `zcompile-all` utility to automatically recompile modified configuration scripts.
- **Native FZF Integration**:
  - Uses `eval "$(fzf --zsh)"` to avoid bloated legacy wrapper plugins.
  - File and directory searches default to `fd` (`--strip-cwd-prefix --hidden --follow --exclude .git`).
  - Previews use `bat` for files and `eza` for directories (`--preview-window right:60%:hidden:wrap`), hidden by default to eliminate I/O overhead upon opening and toggled via `ctrl-/`.
- **Interactive Tab Completion (`fzf-tab`)**:
  - Replaces Zsh's standard completion selection menu with interactive FZF search.
  - Dynamically fuzzy-filters commands, subcommands, flags/options, git branches, processes, environment variables, and filesystem paths.
  - Contextual previews: files (`bat`), directories (`eza`), command and flag descriptions (`$desc`), git history (`git log`), processes (`ps`), and system services (`systemctl`).
  - Navigational shortcuts: `,` and `.` to cycle candidate groups, `/` for continuous path completion into subdirectories, and `Ctrl-Space` for multi-selection.
  - Strict load order: loads immediately after `compinit` and `fzf --zsh`, and prior to widget wrappers (`zsh-autosuggestions`, `fast-syntax-highlighting`).
- **SQLite-Backed Shell History (`atuin`)**:
  - Replaces plaintext `Ctrl-R` (`~/.zsh_history`) with an embedded SQLite database.
  - Sub-millisecond indexed queries across 500k+ commands with execution directory scoping (`filter_mode = "directory"`), exit codes, durations, and fuzzy matching.
  - Configured strictly offline with zero telemetry or remote synchronization (`auto_sync = false`, `update_check = false`).
- **Strict Configuration Separation (`.zshrc` vs `.base.zsh` vs `.tools.sh`)**:
  - `~/.zshrc`: Minimal 6-line loader exclusively sourcing `~/.base.zsh` then `~/.tools.sh`. Must NEVER contain inline tool setups, PATH mutations, or installer boilerplate.
  - `~/.base.zsh`: Core shell configuration, completion caching, prompt initialization (`starship init zsh`), editor variables, and interactive shell widget bindings (such as `eval "$(atuin init zsh)"` and FZF widgets).
  - `~/.tools.sh`: Development runtimes, tool paths, and package manager environments (`pnpm` / `PNPM_HOME`, `fnm`, `pyenv`, `go`, `cargo`, `. "$HOME/.atuin/bin/env"`).
- **Lean Starship Prompt**:
  - Compact 2-line layout with minimal execution overhead (`cmd_duration` threshold 2000ms).
  - Cloud modules (`aws`, `gcloud`, `package`) are disabled to prevent synchronous blocking on network or slow runtimes.
  - Includes `custom.agent_workspace` (`🤖 AGENTS`) to highlight directories governed by `AGENTS.md` and AI engineering rules.

### Diagnostic & Performance Architecture (Ubuntu / Ghostty / Wayland)

Documented root causes and mitigations for hardware/display edge cases:

1. **FPS Drop & Lag Accumulation on Window Resize (Ubuntu Wayland + Nouveau)**:

   - **Hardware Profile**: NVIDIA Kepler GPU (e.g. `Quadro K3100M`) on open-source `nouveau` driver under `ubuntu:GNOME` (Wayland / Mutter) with GTK 4.14.
   - **Root Cause**: The `nouveau` kernel driver on Kepler GPUs has hardware re-clocking disabled (runs at low idle clock ~135MHz) and buggy DRI3/Wayland buffer swapchain sync (`EGL_ANDROID_native_fence_sync`). In GTK 4.14, every window resize triggers dynamic surface reallocations; Nouveau fails to retire stale framebuffers promptly, creating fence wait stalls in Mutter. This throttles frame callbacks down to 1–5 FPS. The issue also manifests in hardware-accelerated Chromium/Brave instances. Reopening the window resets Mutter's Wayland surface and frame queue, restoring performance temporarily.
   - **Mitigation**:
     - Using `herdr` avoids redundant nested PTY redraw overhead.
     - For Kepler Nouveau GPUs, logging into **GNOME on Xorg (X11)** avoids the Wayland presentation stall completely.
     - If remaining on Wayland, setting `window-vsync = false` in `files/ghostty/config` prevents blocking on compositor sync fences.

2. **Ghostty Sudden Crashes in Xfce / Legacy Nested Multiplexers via SSH**:
   - **Root Cause**: Nested terminal multiplexer sessions over SSH with `allow-passthrough on` and `set-clipboard on` (OSC 52) combined with Ghostty's `copy-on-select = clipboard`. Rapid mouse wheel scrolling or selection drags over SSH emit split escape sequences across network packets, triggering X11 selection ownership race conditions (`BadWindow` / `XIOError` in libX11) or unhandled terminal escape parser panics in older builds. Migrating exclusively to Herdr eliminates this nested multiplexer friction.

### Modern CLI Tooling Stack & Non-Destructive Alias Architecture

This environment deploys high-performance, single-binary modern CLI tools and strictly adheres to a **non-destructive alias policy**:

#### 1. Tool Replacements

| Domain             | Classic Tool                        | Modern Replacement         | Rationale & Capabilities                                                                                                                                                                              |
| :----------------- | :---------------------------------- | :------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **AST Diffing**    | `wdiff` / `diff`                    | **`difftastic` (`difft`)** | Rust-based structural diffing using Tree-sitter. Compares programming syntax and AST trees rather than linear text; ignores cosmetic formatting, reordering, and whitespace.                          |
| **Git Pager**      | `diff` / `less`                     | **`git-delta` (`delta`)**  | Syntax-highlighting, side-by-side terminal diff viewer with inline character-level changes and Git integration.                                                                                       |
| **Universal SQL**  | `litecli` / `pgcli`                 | **`usql`**                 | Single Go static binary supporting PostgreSQL, SQLite3, MySQL, Redis, SQL Server, ClickHouse, and Snowflake with auto-completion; eliminates slow Python startups, virtualenvs, and pip dependencies. |
| **Practical Help** | `tldr` (npm/Python)                 | **`tealdeer` (`tldr`)**    | Native Rust implementation of `tldr`. Cache updates take seconds and command responses are instant without Node.js/Python overhead.                                                                   |
| **Token Killer**   | Raw terminal output                 | **`rtk`**                  | Rust proxy compressing LLM command outputs by up to 90%.                                                                                                                                              |
| **Shell History**  | Plaintext `Ctrl-R` (`.zsh_history`) | **`atuin`**                | Embedded SQLite-backed history engine with directory scoping, exit status, duration tracking, and instant fuzzy/prefix search; offline and private.                                                   |

#### 2. Non-Destructive Alias Architecture

Standard Unix utilities (`grep`, `find`, `cat`, `less`) **MUST NEVER** be aliased to modern tools (`rg`, `fd`, `bat`):

- **POSIX & Flag Incompatibility**: Flags like `grep -r` (which in `rg` means `--replace`) or `find . -name "*.go"` (which `fd` rejects with syntax errors) break automated scripts, piped filters, and shell functions.
- **SSH & Muscle Memory Preservation**: Aliasing standard utilities erodes engineer muscle memory, creating friction and failure when SSHing into clean, unprovisioned production servers.
- **Intelligent Strategy**:
  1. _Preserve POSIX Purity_: `grep`, `find`, `cat`, `less` remain completely untouched standard commands.
  2. _Package Normalization_: In Debian/Ubuntu where binaries have divergent names, aliases normalize them to canonical names (`alias fd='fdfind'`, `alias bat='batcat'`).
  3. _Ergonomic Non-Colliding Shortcuts_: Dedicated shortcuts provide modern features without hijacking standard commands:
     - `rgh='rg --hidden'` (ripgrep with hidden files)
     - `fdh='fd --hidden'` (fd with hidden files)
     - `preview='bat'` (syntax-highlighted inspection)
     - `dft='difft'` (Tree-sitter structural diff)
     - `l='eza --icons --classify=always'`, `ll`, `la`, `lt` (modern file listing)

#### 3. Optimal Multi-Platform Installation Strategy

This repository enforces the most efficient installation mechanism per platform. It **strictly prohibits slow packaging systems (`snap`, `flatpak`), heavily outdated distribution packages, and unoptimized compilation**. It **prioritizes official pre-compiled static release binaries** (especially standalone `musl` binaries for Rust and statically-linked Go binaries) for near-instant, zero-dependency provisioning, falling back to optimized source compilation or native Homebrew bottles (on macOS) when appropriate:

| Tool             | Linux (Debian/Ubuntu/Fedora/Arch)                               | macOS (Apple Silicon / Intel)                             | Installation Method & Rationale                                                                                                                                         |
| :--------------- | :-------------------------------------------------------------- | :-------------------------------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **`ghostty`**    | **Compiled from source** (Zig) / Official release package       | **Compiled from source** / Official pre-built bundle      | Built via Zig (`-Doptimize=ReleaseFast`) into `/usr/bin/ghostty` or `/Applications/Ghostty.app`. Avoids all container/snap overhead.                                    |
| **`herdr`**      | **Statically linked binary** (`herdr.dev/install.sh`)           | **Statically linked binary** (`herdr.dev/install.sh`)     | Official static binary into `~/.local/bin/herdr` with zero libc runtime dependencies.                                                                                   |
| **`atuin`**      | **Official standalone binary** (`setup.atuin.sh`)               | **Homebrew bottle** (`brew install atuin`) / `setup.sh`   | Installed via `curl --proto '=https' --tlsv1.2 -LsSf https://setup.atuin.sh \| sh`; avoids 10m+ Rust compilation while embedding SQLite engine.                         |
| **`rtk`**        | **Official pre-compiled release** (`rtk-ai/rtk/install.sh`)     | **Official pre-compiled release** / `cargo` fallback      | Installed via `curl -fsSL https://raw.githubusercontent.com/rtk-ai/rtk/refs/heads/master/install.sh \| sh` into `~/.local/bin/rtk`; cargo compilation used as fallback. |
| **`difftastic`** | **Official pre-compiled release** (GitHub tarball)              | **Homebrew bottle** (`brew install difftastic`)           | Downloads `difft-{version}-x86_64-unknown-linux-gnu.tar.gz` to `~/.local/bin/difft`. Eliminates ~10m compilation of 80 tree-sitter C parsers. Cargo fallback.           |
| **`git-delta`**  | **Official pre-compiled `musl` release** (GitHub tarball)       | **Homebrew bottle** (`brew install git-delta`)            | Downloads `delta-{version}-x86_64-unknown-linux-musl.tar.gz` directly into `~/.local/bin/delta`. Statically linked, cargo fallback.                                     |
| **`tealdeer`**   | **Official pre-compiled `musl` static binary** (GitHub release) | **Homebrew bottle** (`brew install tealdeer`) / Cargo     | Downloads standalone `tealdeer-linux-x86_64-musl` directly to `~/.local/bin/tldr`. Zero dependencies, sub-millisecond execution.                                        |
| **`usql`**       | **Official pre-compiled binary** (GitHub release tarball)       | **Homebrew bottle** (`brew install xo/xo/usql`) / Release | Downloads `usql-{version}-linux-amd64.tar.bz2` into `~/.local/bin/usql` with universal database drivers built-in. Bypasses Go toolchain bootstrap.                      |
| **`fx`**         | **Official pre-compiled static binary** (GitHub release)        | **Homebrew bottle** (`brew install fx`) / Release binary  | Single executable downloaded to `~/.local/bin/fx` for terminal JSON processing.                                                                                         |
| **`lazygit`**    | **Official pre-compiled release** (GitHub tarball)              | **Homebrew bottle** (`brew install lazygit`) / Release    | Extracted directly into `~/.local/bin/lazygit`. Instant updates without Go compilation.                                                                                 |
| **`stylua`**     | **Official pre-compiled release** (GitHub zip)                  | **Homebrew bottle** (`brew install stylua`) / Release     | Extracted directly into `~/.local/bin/stylua`. Bypasses Rust toolchain build overhead.                                                                                  |

---

## 3. Shell Scripting Conventions

All shell scripts in this repository must adhere to the following strict conventions:

1. **Repository-Root Anchoring Loop**:
   Top-level scripts must begin with this loop to ensure consistent execution regardless of invocation CWD:

   ```bash
   while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
       if [[ "$PWD" == "/" ]]; then
           echo "Repository root not found"
           exit 1
       fi
       if ! cd ..; then
           echo "An error occurred while doing 'cd ..'"
           exit 1
       fi
   done
   ```

2. **Sourcing Protocol & `_` Prefix**:

   - Files prefixed with `_` (e.g. `src/remotes/_vars_colors.sh`, `src/remotes/_required_commands.sh`, `src/_install_requirements.sh`, `src/remotes/_cd.sh`) are **libraries/helpers**.
   - They MUST be sourced using `source <path>` or `. <path>`, never executed directly as standalone binaries.

3. **Error Trapping & Robustness**:
   - Scripts that perform side effects must enable errexit and trap errors:
     ```bash
     set -o errexit
     trap exit-error-message ERR SIGINT
     ```
   - Standardized `exit-error-message` functions output high-visibility error blocks to stderr and terminate cleanly.

---

## 4. AI Tooling & Context Architecture: CodeGraph, Engram & RTK

Agents operating in this workspace have access to three specialized AI engineering tools: **CodeGraph**, **Engram**, and **RTK**.

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                              AI Agent Core                              │
└──────────────┬───────────────────────────┬──────────────────────────────┘
               │                           │               │
       (AST & Call Graph)         (Persistent Memory)  (Token Optimizer)
               ▼                           ▼               ▼
     ┌───────────────────┐       ┌───────────────────┐ ┌─────────────┐
     │ CodeGraph MCP     │       │    Engram MCP     │ │   RTK CLI   │
     │ codegraph_explore │       │ mem_save / search │ │  PreToolUse │
     └─────────┬─────────┘       └─────────┬─────────┘ └──────┬──────┘
               │                           │                  │
       [Indexed Symbols]          [Architecture Specs] [Compressed IO]
       - Lua / LazyVim            - SDD Artifacts      - Up to 90%
       - GNOME Shell JS           - Testing Matrix     - Bash Output
       - Config Schemas           - Skill Registry     - Filtered
```

### CodeGraph (`codegraph_explore`)

CodeGraph maintains a SQLite knowledge graph of AST symbols, edges, and call paths across 30+ languages (including Lua and JavaScript).

- **Primary Exploration Tool**:
  - **Always call `codegraph_explore` FIRST** before reading files or running grep loops.
  - Queries accept natural language, symbol names, or flow endpoints:
    - Example symbol lookup: `codegraph_explore(query: "is_path_excluded exclusions.lua")`
    - Example flow exploration: `codegraph_explore(query: "rcmd_backend_gnome Trigger")`
- **Read-Equivalent Contract**:
  - The output returned by `codegraph_explore` contains **verbatim, line-numbered, current on-disk source code**.
  - **Do NOT re-read files** whose code was already returned in a `codegraph_explore` response.
- **Index Exclusions (`codegraph.json`)**:
  - Docs (`docs/**`) and heavy keybinding maps (`files/cursor/keybindings.json`, `files/vscode/keybindings.json`) are intentionally excluded from the index.

### Engram (Project Memory & SDD Persistence)

Engram is the long-term memory engine (`.engram/config.json`, project: `env`):

- **CLI Access**: `/home/arthurnavah/.local/bin/engram`
  - Save: `engram save <title> <content> --project env --topic <topic_key> --type <type>`
  - Context: `engram context env`
  - Stats: `engram stats --project env`
- **MCP Access**: `mem_search`, `mem_get_observation`, `mem_save`
- **Reserved Canonical Topic Keys**:
  - Project Architecture: `sdd-init/env`
  - Testing Capabilities: `sdd/env/testing-capabilities`
  - Skill Registry: `skill-registry`
  - SDD Workflow Artifacts: `sdd/{change-name}/{artifact-type}` (`proposal`, `spec`, `design`, `tasks`, `verify-report`, `archive-report`)
- **Deterministic 2-Step Recovery Protocol**:
  1. `mem_search(query: "topic_key", project: "env")` -> returns observation ID.
  2. `mem_get_observation(id)` -> retrieves complete content.

### RTK (Rust Token Killer) — CLI Output Compression

`rtk` ([github.com/rtk-ai/rtk](https://github.com/rtk-ai/rtk)) is a high-performance CLI proxy compiled in Rust (`~/.cargo/bin/rtk` and `~/.local/bin/rtk`) that filters and compresses command outputs before they enter the LLM context, cutting up to **90% of bash output tokens**.

- **Agent Integration**:
  - Integrated via `rtk init --agent antigravity` and `rtk init -g --auto-patch`.
  - Configures `.agents/plugins/rtk/hooks.json` to intercept `run_command` via `PreToolUse -> rtk hook antigravity`.
  - Transparently compresses outputs for `git`, `cargo`, `npm`, `pnpm`, `pytest`, `go test`, `docker`, and linting utilities.
- **Key Commands & Controls**:
  - `rtk gain`: Displays the live token savings and compression metrics dashboard.
  - `rtk <command>`: Manually run any supported tool through the token killer filter (e.g. `rtk git status`).
  - `rtk proxy <command>`: Raw escape hatch if a command's filtered output is ever garbled or contradictory.
- **Configuration & Telemetry**:
  - Configured at `~/.config/rtk/config.toml` (template in `files/ai/rtk/config.toml`).
  - Telemetry is explicitly disabled (`RTK_TELEMETRY_DISABLED=1`, `telemetry.enabled = false`).

---

## 5. Low-Token AI Engineer Workflow Directives

To maximize token economy, speed, and accuracy, AI agents MUST follow these seven directives:

1. **AST & Symbol Lookups Over Raw Reading**:

   - Never load large files (>150 lines) to find a function or check how an API works.
   - Run `codegraph_explore` with the symbol name. You get the exact implementation and blast radius in a single capped call.

2. **Targeted Line-Range Slicing**:

   - When viewing non-indexed files (e.g. bash scripts or markdown docs), ALWAYS use `StartLine` and `EndLine` parameters with small windows (<80 lines). Never read full files blindly.

3. **Zero-Token Keybinding & Doc Pollution**:

   - `files/cursor/keybindings.json` and `files/vscode/keybindings.json` contain thousands of lines of keymaps. NEVER read or dump these files into chat or context.
   - Do not crawl `docs/**` unless the explicit user instruction is to update documentation.

4. **Targeted Surgical Edits**:

   - Use `replace_file_content` with precise context chunks. Never use `write_to_file` to rewrite an entire file when modifying a few lines or functions.

5. **No Grep Redundancy**:

   - Trust the results of `codegraph_explore`. Do NOT run redundant `grep` passes to re-verify what the AST graph already resolved.

6. **Skill Delegation via Exact Paths**:

   - Read `.atl/skill-registry.md` as an index.
   - When delegating to sub-agents, pass the exact file path under `## Skills to load before work` (e.g. `/home/arthurnavah/.gemini/skills/work-unit-commits/SKILL.md`). Do not generate verbose summaries of the skills.

7. **Transparent RTK CLI Execution**:
   - Prefer running verbose CLI inspections and commands through `rtk` (e.g. `rtk git status`, `rtk cargo check`, `rtk npm test`).
   - The Antigravity plugin hook automatically intercepts `run_command` via `PreToolUse`, eliminating massive log dumps and reducing terminal payload tokens by up to 90%.

---

## 6. Testing, Quality Assurance & Linters

### Strict TDD Status: `false`

There is no workspace-wide unit test runner (e.g. pytest or go test) for the repository as a whole. Do not attempt to run a nonexistent test suite.

### Verification Tools

| Domain            | Tool            | Configuration                       | Check Command                                               |
| ----------------- | --------------- | ----------------------------------- | ----------------------------------------------------------- |
| **Shell Scripts** | `shellcheck`    | `.shellcheckrc`                     | `shellcheck install.sh uninstall.sh utils/*.sh src/**/*.sh` |
| **Lua (Neovim)**  | `stylua`        | `files/nvim/stylua.toml`            | `stylua --check files/nvim/`                                |
| **Go Tools**      | `golangci-lint` | `files/golangci-lint/.golangci.yml` | `golangci-lint run`                                         |
| **JavaScript**    | `eslint`        | `files/eslint/.eslintrc.json`       | `npx eslint files/gnome/extensions/`                        |
| **Formatting**    | `prettier`      | `files/prettier/.prettierrc.json`   | `prettier --check "**/*.{json,md,yaml}"`                    |
| **Markdown**      | `markdownlint`  | `.markdownlint.json`                | `markdownlint "**/*.md"`                                    |

Before committing or completing any change, run the relevant domain linter to verify zero regressions.
