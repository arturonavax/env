# 🦊 arturonavax/env (`anx`)

High-performance, modular dotfiles repository and multi-platform development environment powered by the **`anx`** unified Go CLI/TUI orchestrator.

Designed for software engineers, architects, and AI developers who demand near-instant startup times, rock-solid stability, declarative configurations, and ergonomic command-line workflows across Linux, macOS, and Windows.

---

## Table of Contents

1. [Architectural Principles & Philosophy](#1-architectural-principles--philosophy)
2. [Quick Start & Installation](#2-quick-start--installation)
   - [Unix Bootstrap (`install.sh` / `scripts/bootstrap_anx.sh`)](#unix-bootstrap)
   - [Windows Bootstrap (`install.bat` / `scripts/bootstrap_anx.ps1`)](#windows-bootstrap)
   - [Interactive Bubble Tea v2 TUI Dashboard & Wizard](#interactive-tui-dashboard--wizard)
   - [Autonomous Headless Provisioning & Dry-Run Planning](#headless-provisioning--dry-run-planning)
3. [Declarative Zero-Go-Edit Manifest Architecture](#3-declarative-zero-go-edit-manifest-architecture)
4. [The `anx` Command Hierarchy](#4-the-anx-command-hierarchy)
5. [Decoupled Personal Tools Subsystem & AI Suite](#5-decoupled-personal-tools-subsystem--ai-suite)
   - [Path Isolation Architecture](#path-isolation-architecture)
   - [Tool Lifecycle Updaters](#tool-lifecycle-updaters)
   - [AI Suite (`aitools`) Integration & Ergonomic Shortcuts](#ai-suite-integration--shortcuts)
6. [Lifecycle Memory, Deprecation & Cleanup Engine](#6-lifecycle-memory-deprecation--cleanup-engine)
   - [State Tracking & Orphan Detection](#state-tracking--orphan-detection)
   - [Declarative Archives Registry](#declarative-archives-registry)
   - [The `anx cleanup` / `anx prune` Command](#the-anx-cleanup--prune-command)
7. [Offline Man Pages & Shell Autocompletion](#7-offline-man-pages--shell-autocompletion)
8. [Workstation Hardware Hacks & Security Optimizer](#8-workstation-hardware-hacks--security-optimizer)
9. [Subsystems & Tooling Stack](#9-subsystems--tooling-stack)
10. [Modern CLI Tool Replacements & Non-Destructive Aliases](#10-modern-cli-tool-replacements--non-destructive-aliases)

---

## 1. Architectural Principles & Philosophy

- **Zero-Friction Performance**: Sub-millisecond terminal startup. Startup scripts (`.zshrc`, `.base.zsh`, `.tools.sh`) are compiled into `.zwc` word-code binaries via `zcompile` and memory-mapped by Zsh. Completion caches (`compinit -C`) bypass costly disk audits if validated within 24 hours.
- **Declarative Zero-Go-Edit Engine**: All packages, configurations, profiles, tools, and decommissioned archives are declared in YAML under `manifests/`. Modify behavior without touching Go code.
- **Strict Path Isolation**: Personal developer tools (`ai`, `rtk`, `herdr`) are installed to an isolated directory (`~/.local/share/anx/bin` on Unix, `%LOCALAPPDATA%\anx\tools\bin` on Windows) to prevent collisions with system binaries in `~/.local/bin/`.
- **Environment Memory & Deprecation Tracking**: Tracks historical state in `state.json` with advisory file locking (`flock`). Automatically detects orphaned packages, deprecated components, and legacy archive remnants (e.g. `tmux`), providing safe pruning via `anx cleanup`.
- **Non-Destructive Alias Architecture**: Standard Unix utilities (`grep`, `find`, `cat`, `less`) **are NEVER hijacked or aliased** to modern tools (`rg`, `fd`, `bat`). POSIX purity and muscle memory for remote SSH production environments are strictly preserved, while dedicated shortcuts (`rgh`, `fdh`, `preview`, `dft`, `l`) provide ergonomic speed.
- **True 3-Way Dotfile Reconciliation**: Dotfiles are deployed using Myers unified diffs, ancestor tracking for true 3-way merge, and non-destructive `# >>> arturonavax/env >>>` SmartAppend marker preservation.
- **Offline & Private**: Embedded SQLite shell history (`atuin`) with directory scoping and zero remote sync or telemetry. Offline manual pages (`man anx`) generated directly from Cobra definitions.

---

## 2. Quick Start & Installation

### One-Line Remote Installation (curl / PowerShell)

Bootstrap directly from the `env.arturonavax.dev` subdomain:

```sh
# Linux, macOS, BSD, Raspberry Pi & Android Termux (pure POSIX sh)
curl -fsSL https://env.arturonavax.dev/install.sh | sh

# Shorthand via Cloudflare Edge Worker
curl -fsSL https://env.arturonavax.dev | sh

# Autonomous unattended provisioning
curl -fsSL https://env.arturonavax.dev/install.sh | sh -s -- -y --profile developer

# Safe dry-run preview
curl -fsSL https://env.arturonavax.dev/install.sh | sh -s -- --plan
```

```powershell
# Windows (Native PowerShell)
irm https://env.arturonavax.dev/install.ps1 | iex
```

See [docs/remote_install.md](docs/remote_install.md) and [docs/cloudflare_worker.md](docs/cloudflare_worker.md) for full subdomain configuration and Cloudflare Edge Worker deployment.

### Local Unix Bootstrap

Bootstrap the environment on Linux or macOS. The bootstrap script automatically checks system prerequisites and guarantees Go `>= 1.24` using a resilient 3-tier strategy (**Tier 1 GVM priority** -> **Tier 2 standalone Go release** -> **Tier 3 native package manager**), compiles `bin/anx`, and launches the interactive setup wizard:

```sh
# Clone repository
git clone https://github.com/arturonavax/env.git ~/repos/github.com/arturonavax/env
cd ~/repos/github.com/arturonavax/env

# Interactive bootstrap (compiles anx and launches TUI wizard)
./install.sh

# Autonomous non-interactive installation
./install.sh -y
```

### Windows Bootstrap

Run the native PowerShell preflight and bootstrap script via `install.bat` or `scripts/bootstrap_anx.ps1`. Automatically downloads the official SHA-256 verified Go archive from `go.dev` if missing, compiles `anx.exe` and `aitools.exe` into `%LOCALAPPDATA%\Programs\anx\bin`, and registers user PATH:

```cmd
install.bat
```

### Interactive TUI Dashboard & Wizard

Run the full-screen terminal user interface powered by [Bubble Tea v2](https://github.com/charmbracelet/bubbletea):

```sh
# Interactive full-screen workstation dashboard (8 tabs, keyboard/mouse navigation)
anx dashboard

# Step-by-step interactive configuration & package selection wizard
anx wizard
```

### Headless Provisioning & Dry-Run Planning

For automated CI/CD pipelines, container builds, or previewing changes without touching the filesystem:

```sh
# Dry-run execution plan for the active workstation profile
anx plan

# Preview execution plan for a specific profile (minimal, developer, full, etc.)
anx plan --profile minimal

# Apply configuration autonomously
anx apply --yes --profile developer
```

### Multi-Platform System Uninstallation & Teardown

Cleanly remove compiled binaries, system PATH modifications, and caches, with optional configuration purging:

```sh
# Unix / macOS teardown
./uninstall.sh --plan           # Dry-run preview of components to be removed
./uninstall.sh                  # Interactive teardown
./uninstall.sh -y --purge       # Autonomous teardown including configuration directories

# Windows teardown
uninstall.bat                   # Native entrypoint delegating to scripts\uninstall_anx.ps1
uninstall.bat -Plan             # Preview dry-run on Windows
uninstall.bat -Yes -PurgeConfigs

# Unified CLI teardown from anx itself
anx uninstall --system          # Teardown platform binaries and system integrations
anx uninstall --system --purge  # Complete purge including ~/.config/anx
```

---

## 3. Declarative Zero-Go-Edit Manifest Architecture

All packages, dotfiles, profiles, tools, and decommissioned archives are declared in modular YAML manifests embedded directly in the `anx` binary via `//go:embed`, with real-time overlay support:

```text
manifests/
├── packages/           # 80 packages with multi-distro translations & fallbacks
│   ├── base.yaml       # Core CLI utilities (curl, git, zsh, ripgrep, fd, bat, eza)
│   ├── devtools.yaml   # Language runtimes, compilers, and debuggers
│   ├── database.yaml   # Universal SQL tools (usql)
│   ├── editors.yaml    # Neovim, LazyVim dependencies, Mason runtimes
│   └── terminal.yaml   # Ghostty, Herdr, Atuin, Starship
├── hacks/              # Machine- and hardware-specific system optimizations & scripts
│   ├── dell-precision-m6800/ # Dell M6800 fan, GPU and power optimizations
│   └── x11-scroll-lock-led/  # Hardware Scroll Lock LED control
├── configs.yaml        # 23 managed dotfile configuration assets (files/)
├── profiles.yaml       # 8 workstation profiles with multi-level inheritance
├── hacks.yaml          # Hardware optimization declarations and DMI matching rules
├── tools.yaml          # Decoupled developer and personal tools registry
└── archives.yaml       # Decommissioned legacy components (tmux, wdiff, litecli)
```

### User Drop-In Overlays
You can add or override manifests without touching the repository or recompiling Go:
- Drop custom package/config manifests into `~/.config/anx/manifests.d/*.yaml`
- Drop custom decoupled tool declarations into `~/.config/anx/tools.d/*.yaml`

---

## 4. The `anx` Command Hierarchy

`anx` organizes commands into clean, logical groups with full `--help`, tab autocompletion, and JSON/YAML machine outputs:

```text
Packages & Profiles:
  apply       Execute the plan to bring this machine to the desired state
  info        Show a package definition, strategies and resolved install source
  install     Install packages (and their declared dependencies)
  list        List catalog packages with their installation state (--missing|--deprecated|--orphaned)
  plan        Display the dry-run execution plan for the desired state
  profile     Inspect and select configuration profiles
  resume      Resume an interrupted apply (completed steps are skipped idempotently)
  search      Search the declarative catalog by id, name, category or description
  uninstall   Remove managed packages

Dotfiles & Backups:
  backup      Create, list and restore configuration snapshots (50MB quota rotation)
  capture     Ingest live dotfiles into repository with optional git commit & push
  diff        Show real differences between repository and live dotfiles
  history     Display the transaction journal
  rollback    Restore the most recent pre-change configuration snapshot
  sync        Deploy dotfiles with true 3-way merge (or --push to capture)

System, Hacks & Maintenance:
  cleanup     Safely detect and prune orphaned, deprecated, and archived components (alias: prune)
  hacks       Machine-specific hardware/OS optimizations (opt-in, never applied implicitly)
  optimize    Audit hardware/kernel/security and produce a guardrail-validated optimization script
  rebuild     Rebuild global tools after toolchain upgrades (Go, Rust, Node, Python, Neovim)
  update      Inspect repository, personal tools, and dotfiles for available updates
  upgrade     End-to-end upgrade: pull repo, update tools, apply package plans, and sync dotfiles

Personal Tools & AI Suite:
  ai          AI productivity, rolling quota management, and workflow automation suite
  llm         Query AI models non-interactively (shortcut for 'anx tools ai llm')
  tools       Manage, execute, and update decoupled developer tools and AI suite

Diagnostics & Meta:
  alias       Create custom binary command alias/symlink (e.g. for forks or personal naming)
  audit       Scan installed tools for known vulnerabilities (OSV.dev)
  config      Show effective anx configuration and paths
  dashboard   Launch the full-screen interactive dashboard
  doc         Generate documentation, man pages (doc man), and markdown references (doc markdown)
  doctor      Run system health checks and diagnostics
  status      Inspect platform, packages and sync state
  theme       List and persist UI themes (tokyo-night, catppuccin, gruvbox, dracula, etc.)
  version     Print version information
  wizard      Interactive environment setup wizard (use --yes for autonomous mode)

Additional Commands:
  completion  Generate or install shell completion script (--install for bash|zsh|fish|powershell)
```

---

## 5. Decoupled Personal Tools Subsystem & AI Suite

The personal tools subsystem decouples developer utilities from the `anx` core binary. Tools can be written in any language (**Go, Rust, Python, Bash, Node.js**) and updated independently without recompiling `anx`.

### Path Isolation Architecture

To avoid namespace collisions with generic system packages in `~/.local/bin/`, personal tools are isolated to:
- **Unix**: `~/.local/share/anx/bin` (or `$XDG_DATA_HOME/anx/bin`)
- **Windows**: `%LOCALAPPDATA%\anx\tools\bin`
- **Override**: `ANX_TOOLS_BIN` environment variable

System utilities like `difftastic`, `delta`, and `tealdeer` declare `allow_global_path: true` to discover system package manager installs, while personal tools remain strictly sandboxed.

### Tool Lifecycle Updaters

Update decoupled tools using their declarative update strategies:

```sh
# List all registered personal and external tools
anx tools list

# Inspect runtime status and execution target for a tool
anx tools info rtk

# Execute a tool through the decoupled runner
anx tools run rtk --version

# Update a specific tool according to its declared strategy
anx tools update rtk
```

Supported update strategies:
- `self_update`: Executes tool's built-in updater (e.g. `rtk update`, `herdr upgrade`).
- `package`: Delegates to `anx install <package>` with DAG dependency resolution.
- `anx_self`: Handled by `anx` multicall binary dispatch.
- `command`: Executes declarative custom update commands.

### AI Suite Integration & Shortcuts

`aitools` is fully ported to Go and integrated natively under `anx tools ai`:

```sh
# AI Conventional Commit generation
anx tools ai commit      # Shortcut: anx tools commit, alias: ai-commit / ai-cm

# AI Pull Request description generation
anx tools ai pr          # Shortcut: anx tools pr

# Direct non-interactive LLM query
anx tools ai llm "query" # Shortcut: anx tools llm, top-level alias: anx llm

# 5-hour rolling quota telemetry inspection
anx tools ai quota       # Shortcut: anx tools quota

# Motherboard 04:00 AM RTC hardware alarm wakeup daemon
anx tools ai daemon
```

---

## 6. Lifecycle Memory, Deprecation & Cleanup Engine

### State Tracking & Orphan Detection

Every resource provisioned by `anx` is tracked in `~/.local/state/anx/state.json` alongside ancestor file contents for 3-way merges. When you switch profiles or remove a component from manifests, `anx` remembers that the resource was installed and surfaces it as **Orphaned**.

### Declarative Archives Registry

Decommissioned components are tracked historically in `manifests/archives.yaml`:
- **`tmux`**: Decommissioned in favor of [Herdr](https://herdr.dev/) for native agent detection and zero nested PTY redraw friction.
- **`wdiff`**: Replaced by [Difftastic](https://github.com/Wilfred/difftastic) for AST Tree-sitter structural diffing.
- **`litecli`**: Replaced by [usql](https://github.com/xo/usql) eliminating Python virtualenv overhead.

### The `anx cleanup` / `anx prune` Command

Detects and cleans up stale remnants across three memory tiers:

```sh
# Preview all cleanup candidates without modifying anything (dry-run)
anx cleanup --dry-run

# Interactively review and prune stale components
anx cleanup

# Autonomously confirm cleanup of orphaned packages only
anx cleanup --yes --orphans-only
```

Filtering in package listings:
```sh
# View all deprecated packages with their recommended replacements
anx list --deprecated

# View all orphaned packages tracked in state.json
anx list --orphaned
```

---

## 7. Offline Man Pages & Shell Autocompletion

### Standard Troff Man Pages (`man anx`)

Generate and install standard Unix troff section 1 man pages using `github.com/spf13/cobra/doc`:

```sh
# Install man pages directly for the active user (~/.local/share/man/man1)
anx doc man --install

# View manual directly
man anx
```

### Markdown Documentation Generation

Generate GitHub-flavored Markdown reference docs for the entire command hierarchy:

```sh
anx doc markdown --dir docs/cli
```

### Shell Autocompletion

Generate or permanently install tab-completions for your shell:

```sh
# Install completion script directly to shell completions folder
anx completion zsh --install
anx completion bash --install
anx completion fish --install
anx completion powershell --install
```

---

## 8. Workstation Hardware Hacks & Security Optimizer

### Hardware Hacks Registry (`manifests/hacks/` & `manifests/hacks.yaml`)
Workstation-specific hardware and kernel tweaks with DMI hardware auto-detection:
- **`dell-precision-m6800`**: Quad-core i7, NVIDIA Quadro Kepler fan curve overrides, GPU power profiles, and thermal mitigations.
- **`x11-scroll-lock-led`**: Controls physical hardware Scroll Lock LED on modern desktop environments.

```sh
# List and detect hardware optimizations for the current machine
anx hacks detect
anx hacks apply dell-precision-m6800
```

### System Security & Hardware Optimizer (`anx optimize`)
Runs Lynis security audits and profiles CPU, NVMe/SSD, and kernel parameters, enforcing 12 strict automated security guardrails before applying reversible optimization scripts.

---

## 9. Subsystems & Tooling Stack

- **Code Editor Suite (`editor`)**: Neovim v0.12+ with [LazyVim](https://www.lazyvim.org/) and centralized [Exclusions Architecture](files/nvim/lua/config/exclusions.lua) (O(1) hash table with 1,000-entry memoization cache for sub-millisecond LSP/completion performance). VS Code and Cursor IDE settings are retained for legacy compatibility without offering purge.
- **GPU Terminal & Multiplexer (`terminal`)**: [Ghostty](https://ghostty.org/) GPU terminal (primary/recommended) with [Herdr](https://herdr.dev/) agent multiplexer. Also fully supports Kitty, WezTerm, and Alacritty with Tokyo Night palettes; Windows Terminal (with WSL notes); and Warp (noting that custom prompts, FZF widgets, and Herdr are disabled by Warp's closed architecture).
- **Multi-Shell Architecture**: Zsh is the canonical source of truth with `.zwc` word-code compilation and memory-mapped fast loading (`~/.base.zsh` and `~/.tools.sh`). Complete native adapters are provided for Bash (`files/bash/`), Fish (`files/fish/`), and Nushell (`files/nushell/`).
- **Prompt**: [Starship](https://starship.rs/) with compact 2-line layout and AI workspace detection (`🤖 AGENTS`).
- **Window Switcher**: [`rcmd`](files/rcmd/) instant application switcher via Right-Alt key combinations supporting GNOME Shell D-Bus extension ([`rcmd-shell@arturonavax.dev`](files/gnome/extensions/rcmd-shell@arturonavax.dev/)), XFCE (via `files/rcmd/backends/xfce.sh` and `rcmd --sync-xfce`), Hyprland, Sway, KDE KWin, and X11.

---

## 10. Modern CLI Tool Replacements & Non-Destructive Aliases

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
| **AI Commits** | Manual commit | **`anx tools ai commit`** | AI Conventional Commit generation from git diff. | `ai-commit` / `ai-cm` |
| **System Tuning** | Manual tuning | **`anx optimize`** | Hardware & Lynis audit with AI optimization script and guardrails. | `ai-opt` / `sysopt` |
| **Clipboard** | System specific | **`clip` / `clip-paste`** | Multiplatform clipboard wrapper (Wayland, X11, macOS, WSL, Termux). | `cb`, `copy`, `paste` |
| **File Listing** | `ls` | **`eza`** | Modern file listing with icons, git status, and tree view. | `l`, `ll`, `la`, `lt` |
| **File Search** | `find` | **`fd`** | Fast directory traversal ignoring `.git`. | `fdh` (`fd --hidden`) |
| **Code Search** | `grep` | **`ripgrep` (`rg`)** | Ultra-fast recursive regex search ignoring `.git`. | `rgh` (`rg --hidden`) |
| **File Preview** | `cat` / `less` | **`bat`** | Syntax-highlighted file pager with line numbers. | `preview` |

---

## 11. Multi-Platform Automated Testing & Docker E2E Verification

`arturonavax/env` includes a comprehensive multi-platform automated testing harness covering unit tests, integration tests, and containerized end-to-end (E2E) verification:

### Native Go Test Suite
```sh
# Run full unit and integration test suite with race detector
go test -v -race ./...
```

### Hermetic Docker E2E Testing Suite
The Docker E2E test harness (`tests/e2e/`) validates the complete lifecycle inside isolated, multi-stage containers across multiple Linux environments (Debian Bookworm and Alpine Linux):

- **Remote Installer Pipeline**: Tests `curl -fsSL ... | bash` remote execution without pre-installed repositories.
- **GitHub Release Binary**: Tests pre-compiled binary download and multicall dispatch in a pure zero-Go environment.
- **Build From Source**: Verifies Go toolchain bootstrapping and compilation from repository source.
- **CLI & TUI Interactivity**: Tests unattended batch automation (`anx apply -y`) and full Bubble Tea v2 interactive wizard stages via pseudo-terminal emulation (`tests/e2e/scripts/run_wizard_pty.py`).
- **Concurrent Execution**: Validates DAG solver resolution and tool inspections under parallel workloads.
- **Dotfile Reconciliation & Doctor**: Tests Myers 3-way dotfile merge, hierarchical snapshot rotation (50MB quota), personal tools path isolation, and comprehensive diagnostic health checks.

```sh
# Run full E2E suite across both Debian and Alpine
./scripts/test_e2e_docker.sh all

# Or run specific target environments
./scripts/test_e2e_docker.sh debian
./scripts/test_e2e_docker.sh alpine
```

---

## License

MIT © Arturo Nava
