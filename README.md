# arturonavax/env

High-performance, modular dotfiles repository and multi-platform development environment supporting Linux (Debian/Ubuntu, Arch, Fedora), macOS, and Windows (PowerShell/Windows Terminal).

Orchestrated by the unified Go CLI/TUI platform **ANX** (`anx`) and powered by a declarative zero-Go-edit YAML manifest architecture, Bubble Tea v2 full-screen TUI, and decoupled personal tooling engine.

---

## 1. System Architecture

```text
├── cmd/
│   ├── anx/                # Master multicall CLI & TUI entrypoint
│   └── aitools/            # Standalone multicall symlink entrypoint
├── manifests/              # Declarative zero-Go-edit YAML manifests
│   ├── packages/           # 80 packages with multi-distro translations & fallbacks
│   ├── configs.yaml        # 23 managed dotfile configuration assets
│   ├── profiles.yaml       # 8 workstation profiles with multi-level inheritance
│   ├── hacks.yaml          # Declarative hardware optimization definitions
│   ├── hacks/              # Machine- and hardware-specific system optimizations
│   │   ├── dell-precision-m6800/ # Dell M6800 fan, GPU and power optimizations
│   │   └── x11-scroll-lock-led/  # Hardware Scroll Lock LED control
│   ├── tools.yaml          # Decoupled personal tools engine registry
│   └── archives.yaml       # Decommissioned legacy tools (tmux, wdiff, litecli)
├── scripts/                # Multi-platform pre-flight & Go bootstrap scripts
│   ├── bootstrap_anx.sh    # Unix bootstrap (GVM priority -> standalone -> native PM)
│   ├── bootstrap_anx.ps1   # Windows bootstrap (official go.dev archive with SHA-256)
│   └── safe_suspend.sh     # Standalone hardware-safe power management utility
├── internal/               # Production Go Hexagonal Architecture packages
│   ├── domain/             # Pure domain models (plans, resources, transactions)
│   ├── app/                # Application services (Install, Sync, Hacks, Ops, Doctor, Audit)
│   ├── manifest/           # Manifest loader (embedded + overlay) & DAG solver
│   ├── catalog/            # Resource registry
│   ├── providers/          # Dispatcher (native, github_release, script, source, language)
│   ├── configsync/         # Myers unified diff, bidirectional sync & 3-way merge
│   ├── snapshot/           # Hierarchical snapshot manager (50MB quota)
│   ├── hacks/              # DMI/hardware detector & hack runner
│   ├── ops/                # Multi-runtime rebuilder, power manager & Lynis optimizer
│   ├── ai/                 # Go port of aitools (JSONC, harnesses, gitflow, RTC wake)
│   ├── tui/                # Modular Bubble Tea v2 TUI (8 tabs, 4 Golden Rules)
│   └── cli/                # Cobra + Viper CLI commands
├── install.sh              # Primary POSIX sh installation entry point
├── install.bat             # Windows installation entry point
├── uninstall.sh            # Systematic uninstallation and cache purging entry point
└── files/                  # Canonical configuration files deployed to target systems
```

---

## 2. Quick Start & Installation

### One-Line Remote Installation

Bootstrap directly from `env.arturonavax.dev`:

#### Linux, macOS, BSD & Raspberry Pi (POSIX sh)

Interactive guided installation (default in interactive terminal):

```sh
curl -fsSL https://env.arturonavax.dev/install.sh | sh
```

Shorthand via Cloudflare Edge Worker:

```sh
curl -fsSL https://env.arturonavax.dev | sh
```

Autonomous unattended provisioning (zero prompts):

```sh
curl -fsSL https://env.arturonavax.dev/install.sh | sh -s -- -y --profile developer
```

Safe dry-run preview (previews touched files without modifying the system):

```sh
curl -fsSL https://env.arturonavax.dev/install.sh | sh -s -- --plan
```

#### Windows 10 / 11 (PowerShell)

Interactive installation:

```powershell
irm https://env.arturonavax.dev/install.ps1 | iex
```

Autonomous unattended provisioning:

```powershell
& ([scriptblock]::Create((irm https://env.arturonavax.dev/install.ps1))) -Yes -Profile developer
```

#### Windows (Command Prompt - CMD)

```bat
curl -fsSL https://env.arturonavax.dev/install.bat -o %TEMP%\install.bat && %TEMP%\install.bat
```

See [docs/remote_install.md](docs/remote_install.md) and [docs/cloudflare_worker.md](docs/cloudflare_worker.md) for full subdomain configuration and Cloudflare Edge Worker deployment.

---

### Local Unix Bootstrap

Bootstrap the environment on Linux or macOS. The bootstrap script automatically checks system prerequisites and guarantees Go `>= 1.24` using a resilient 3-tier strategy (**Tier 1 GVM priority** -> **Tier 2 standalone Go release** -> **Tier 3 native package manager**), compiles `bin/anx`, and launches the interactive setup wizard:

Clone repository:

```sh
git clone https://github.com/arturonavax/env.git ~/repos/github.com/arturonavax/env
```

Navigate to repository root:

```sh
cd ~/repos/github.com/arturonavax/env
```

Interactive bootstrap (compiles anx and launches TUI wizard):

```sh
./install.sh
```

Autonomous non-interactive installation:

```sh
./install.sh -y
```

---

### Windows Bootstrap

Run the native PowerShell preflight and bootstrap script via `install.bat` or `scripts/bootstrap_anx.ps1`. Automatically downloads the official SHA-256 verified Go archive from `go.dev` if missing, compiles `anx.exe` into `%LOCALAPPDATA%\Programs\anx\bin`, and registers user PATH:

```bat
install.bat
```

---

### Interactive TUI Dashboard & Wizard

Run the full-screen terminal user interface powered by [Bubble Tea v2](https://github.com/charmbracelet/bubbletea):

Interactive full-screen workstation dashboard (8 tabs, keyboard and mouse navigation):

```sh
anx dashboard
```

Step-by-step interactive configuration & package selection wizard:

```sh
anx wizard
```

---

### Headless Provisioning & Dry-Run Planning

For automated CI/CD pipelines, container builds, or previewing changes without touching the filesystem:

Dry-run execution plan for the active workstation profile:

```sh
anx plan
```

Preview execution plan for a specific profile:

```sh
anx plan --profile minimal
```

Apply configuration autonomously:

```sh
anx apply --yes --profile developer
```

---

### Hardware-Safe Power Management & Suspend (`anx sleep`)

Hardware-safe power transitions designed to eliminate GPU driver deadlocks (Nouveau Kepler with `nouveau.pstate=1`) and Xfce video crashes:

Display diagnostic health check of power states, GPU driver, active VT, and swap space:

```sh
anx sleep status
```

Lock desktop session and enter DPMS low-power screen standby (zero crash risk, instant wake):

```sh
anx sleep lock
```

Suspend CPU to idle states without cutting GPU PLL power:

```sh
anx sleep s2idle
```

Deep S3 sleep with automatic VT switch to release X11 DRM master:

```sh
anx sleep deep
```

Suspend to disk (validates Swap >= RAM):

```sh
anx sleep hibernate
```

---

### Multi-Platform System Uninstallation & Teardown

Cleanly remove compiled binaries, system PATH modifications, and caches, with optional configuration purging:

#### Unix / macOS Teardown

Preview uninstallation teardown plan:

```sh
./uninstall.sh --plan
```

Interactive uninstallation:

```sh
./uninstall.sh
```

Autonomous uninstallation including configuration purge:

```sh
./uninstall.sh -y --purge
```

#### Windows Teardown

Preview uninstallation teardown plan on Windows:

```bat
uninstall.bat -Plan
```

Autonomous uninstallation including configuration purge on Windows:

```bat
uninstall.bat -Yes -PurgeConfigs
```

#### Unified CLI Teardown

Teardown platform binaries and system integrations via `anx`:

```sh
anx uninstall --system
```

Complete purge including configurations via `anx`:

```sh
anx uninstall --system --purge
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
  sleep       Hardware-safe power transitions, screen standby via DPMS, and hibernation diagnostics
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

List all registered personal and external tools:

```sh
anx tools list
```

Inspect runtime status and execution target for a tool:

```sh
anx tools info rtk
```

Execute a tool through the decoupled runner:

```sh
anx tools run rtk --version
```

Update a specific tool according to its declared strategy:

```sh
anx tools update rtk
```

Supported update strategies:
- `self_update`: Executes tool's built-in updater (e.g. `rtk update`, `herdr upgrade`).
- `package`: Delegates to `anx install <package>` with DAG dependency resolution.
- `anx_self`: Handled by `anx` multicall binary dispatch.
- `command`: Executes declarative custom update commands.

### AI Suite Integration & Shortcuts

`aitools` is fully ported to Go and integrated natively under `anx tools ai`:

AI Conventional Commit generation from git diff:

```sh
anx tools ai commit
```

AI Pull Request description generation:

```sh
anx tools ai pr
```

Direct non-interactive LLM query:

```sh
anx tools ai llm "query"
```

5-hour rolling quota telemetry inspection:

```sh
anx tools ai quota
```

Hardware RTC auto-wake daemon and power scheduler:

```sh
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

Preview all cleanup candidates without modifying anything (dry-run):

```sh
anx cleanup --dry-run
```

Interactively review and prune stale components:

```sh
anx cleanup
```

Autonomously confirm cleanup of orphaned packages only:

```sh
anx cleanup --yes --orphans-only
```

View all deprecated packages with their recommended replacements:

```sh
anx list --deprecated
```

View all orphaned packages tracked in state.json:

```sh
anx list --orphaned
```

---

## 7. Offline Man Pages & Shell Autocompletion

### Standard Troff Man Pages (`man anx`)

Install man pages directly for the active user (`~/.local/share/man/man1`):

```sh
anx doc man --install
```

View manual directly:

```sh
man anx
```

### Markdown Documentation Generation

Generate GitHub-flavored Markdown reference docs for the entire command hierarchy:

```sh
anx doc markdown --dir docs/cli
```

### Shell Autocompletion

Install completion script directly for Zsh:

```sh
anx completion zsh --install
```

Install completion script directly for Bash:

```sh
anx completion bash --install
```

Install completion script directly for Fish:

```sh
anx completion fish --install
```

Install completion script directly for PowerShell:

```powershell
anx completion powershell --install
```

---

## 8. Workstation Hardware Hacks & Security Optimizer

### Hardware Hacks Registry (`manifests/hacks/` & `manifests/hacks.yaml`)

Workstation-specific hardware and kernel tweaks with DMI hardware auto-detection:
- **`dell-precision-m6800`**: Quad-core i7, NVIDIA Quadro Kepler fan curve overrides, GPU power profiles, and thermal mitigations.
- **`x11-scroll-lock-led`**: Controls physical hardware Scroll Lock LED on modern desktop environments.

Detect hardware optimizations matching the current machine:

```sh
anx hacks detect
```

Apply a specific hardware hack:

```sh
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

Run full unit and integration test suite with race detector:

```sh
go test -v -race ./...
```

### Hermetic Docker E2E Testing Suite

The Docker E2E test harness (`tests/e2e/`) validates the complete lifecycle inside isolated, multi-stage containers across multiple Linux environments (Debian Bookworm and Alpine Linux):

- **Remote Installer Pipeline**: Tests `curl -fsSL ... | sh` remote execution without pre-installed repositories.
- **GitHub Release Binary**: Tests pre-compiled binary download and multicall dispatch in a pure zero-Go environment.
- **Build From Source**: Verifies Go toolchain bootstrapping and compilation from repository source.
- **CLI & TUI Interactivity**: Tests unattended batch automation (`anx apply -y`) and full Bubble Tea v2 interactive wizard stages via pseudo-terminal emulation (`tests/e2e/scripts/run_wizard_pty.py`).
- **Concurrent Execution**: Validates DAG solver resolution and tool inspections under parallel workloads.
- **Dotfile Reconciliation & Doctor**: Tests Myers 3-way dotfile merge, hierarchical snapshot rotation (50MB quota), personal tools path isolation, and comprehensive diagnostic health checks.

Run full E2E suite across both Debian and Alpine:

```sh
./scripts/test_e2e_docker.sh all
```

Run specifically for Debian:

```sh
./scripts/test_e2e_docker.sh debian
```

Run specifically for Alpine:

```sh
./scripts/test_e2e_docker.sh alpine
```

---

## License

MIT © Arturo Nava
