# anx — Hexagonal Architecture & Package Design

## 1. Architectural Style

`anx` is organized as an enterprise-grade **Modular Monolith with Hexagonal Boundaries** (Ports and Adapters). The core business logic is completely isolated from operating system specifics, external package managers, CLI frameworks, and UI libraries.

```
       +-----------------------------------------------------------------------------------+
       |                                Presentation Layer                                 |
       |  +------------------------+  +---------------------+  +------------------------+  |
       |  |   Cobra CLI Commands   |  |  Bubbletea v2 TUI   |  | Multicall aitools CLI  |  |
       |  |     (internal/cli)     |  |   (internal/tui)    |  |     (internal/ai/cli)  |  |
       |  +-----------+------------+  +----------+----------+  +-----------+------------+  |
       +--------------|--------------------------|-------------------------|---------------+
                      |                          |                         |
                      v                          v                         v
       +-----------------------------------------------------------------------------------+
       |                                 Application Layer                                 |
       |                                  (internal/app)                                   |
       |  * InstallService (DAG execution, compensation, rollback barriers)                |
       |  * SyncService    (Myers diff, 3-way merge, SmartAppend, reverse capture)         |
       |  * HacksService   (DMI/GPU hardware probe, declarative matchers, safe apply)      |
       |  * OpsService     (Rebuilder across runtimes, Lynis security/hardware optimizer)  |
       |  * DoctorService  (Platform pre-flight diagnostics, remediations)                 |
       |  * AuditService   (OSV.dev vulnerability scanner, advisory lookup)                |
       +-----------------------------------------+-----------------------------------------+
                                                 |
                                                 v
       +-----------------------------------------------------------------------------------+
       |                                   Domain Layer                                    |
       |                                 (internal/domain)                                 |
       |  * ManagedResource, Package, ConfigurationResource, Profile, HackProfile          |
       |  * ExecutionPlan, PlanStep, StepStatus, CompensationStep                          |
       |  * Transaction, JournalEntry, Snapshot, BackupManifest, AuditFinding              |
       |  * HardwareProfile, HardwareMatcher, MergeConflict, ExitCodes (0..5, 130)         |
       +-----------------------------------------+-----------------------------------------+
                                                 ^
                                                 | (implements ports)
       +-----------------------------------------+-----------------------------------------+
       |                                  Adapters Layer                                   |
       |  * internal/manifest   (Embedded + filesystem YAML manifest loader, DAG solver)   |
       |  * internal/catalog    (Package, config & profile registry)                       |
       |  * internal/providers  (Dispatcher: native PMs, github_release, script, source)   |
       |  * internal/configsync (Myers unified diff, directory deployer, SmartAppend)      |
       |  * internal/snapshot   (File + directory snapshots, 50MB quota, SHA-256 manifest) |
       |  * internal/security   (Safe runner, SHA-256 verifier, OSV cache, script guards)  |
       |  * internal/hacks      (DMI / sysfs / lspci hardware prober & runner)             |
       |  * internal/ai         (JSONC config, multi-harness adapters, gitflow, RTC wake)  |
       |  * internal/journal    (Durable JSONL audit log)                                  |
       |  * internal/theme      (6 palettes: Tokyo Night, Storm, Catppuccin, Gruvbox, etc.)|
       +-----------------------------------------------------------------------------------+
```

## 2. Directory Structure

```text
├── cmd/
│   ├── anx/
│   │   └── main.go                 # Unified entry point: multicall dispatch, exit codes
│   └── aitools/
│       └── main.go                 # Standalone AI suite entry point
├── internal/
│   ├── domain/                     # Pure domain models, zero external dependencies
│   │   ├── audit.go                # AuditFinding, Severity, Advisory
│   │   ├── errors.go               # ExitError, Spec exit codes (0..5, 130)
│   │   ├── hacks.go                # HackProfile, HardwareMatcher, HardwareProfile
│   │   ├── merge.go                # MergeConflict, ThreeWayDiff, DiffResult
│   │   ├── plan.go                 # ExecutionPlan, PlanStep, ActionType
│   │   ├── resource.go             # ManagedResource, ConfigurationResource, Strategy
│   │   ├── snapshot.go             # Snapshot, BackupMetadata, SnapshotItem
│   │   └── transaction.go          # Transaction, JournalEntry, CompensationAction
│   ├── app/                        # Application use cases (Hexagonal Inbound Ports)
│   │   ├── install_service.go      # DAG installation & rollback compensation
│   │   ├── sync_service.go         # Dotfile reconciliation & reverse capture
│   │   ├── hacks_service.go        # Hardware hack discovery & execution
│   │   ├── ops_service.go          # Rebuilder & Lynis optimizer
│   │   ├── doctor_service.go       # System health diagnostics
│   │   └── audit_service.go        # OSV vulnerability scanner
│   ├── manifest/                   # Declarative YAML manifest schema & DAG solver
│   ├── catalog/                    # In-memory registry with embedded & overlay configs
│   ├── providers/                  # Multi-tier provider dispatcher
│   │   ├── dispatcher.go           # Strategy router (native, github, script, source, lang)
│   │   ├── native.go               # apt, dnf, pacman, zypper, apk, brew, pkg, winget
│   │   ├── github_release.go       # GitHub release binary with SHA-256 verification
│   │   ├── script.go               # Verified curl-to-shell installers
│   │   ├── source.go               # Git clone + build steps (e.g. Zig, Make, Cargo)
│   │   └── language_pkg.go         # go install, cargo install, pnpm add -g, pipx
│   ├── configsync/                 # Dotfile synchronization engine
│   │   ├── diff.go                 # Pure-Go Myers unified line diff & tree diff
│   │   ├── sync.go                 # Bidirectional sync & smart-append marker blocks
│   │   └── threeway.go             # True 3-way merge using state.json base hashes
│   ├── snapshot/                   # Snapshot manager & 50MB quota rotation
│   ├── hacks/                      # Declarative hardware detector & hack runner
│   ├── ops/                        # Multi-runtime tool rebuilder & Lynis optimizer
│   ├── ai/                         # Complete Go port of aitools
│   │   ├── config/                 # Pure-Go JSONC parser, deep merge, state store
│   │   ├── harness/                # Multi-harness adapters (agy, claude, codex, etc.)
│   │   ├── contextbuilder/         # File/dir attachment with 500KB cap
│   │   ├── gitflow/                # Tracker-agnostic commit & template-aware PR
│   │   ├── pipeline/               # Declarative custom AI pipelines
│   │   ├── quota/                  # 5-hour rolling usage tracking & watch dashboard
│   │   ├── daemon/                 # D-Bus / RTC 04:00 AM alarm wakeup & safe suspend
│   │   ├── wizard/                 # Bubbletea v2 AI configuration wizard
│   │   └── cli/                    # Cobra command tree builder for AI commands
│   ├── state/                      # Atomic state.json store & anx.lock advisory lock
│   ├── security/                   # Capped runner, SHA-256 verifier, OSV client, guardrails
│   ├── execution/                  # Bounded concurrency engine with errgroup & barriers
│   ├── platform/                   # Multi-platform OS, Arch, Distro, WSL, Termux prober
│   ├── theme/                      # 6 terminal palettes with Lipgloss styles
│   ├── tui/                        # Modular Bubbletea v2 TUI adhering to 4 Golden Rules
│   │   ├── types.go                # TabID enum (8 tabs), Deps, async messages
│   │   ├── styles.go               # Border rendering (-2 deduction), ANSI truncation
│   │   ├── model.go                # TUI state model & concurrent init batch
│   │   ├── update.go               # Message dispatcher
│   │   ├── update_keyboard.go      # Tab cycling, navigation, shortcuts
│   │   ├── update_mouse.go         # Layout-matched X/Y mouse clicks & scroll wheel
│   │   ├── view.go                 # View() returning tea.View with 8 tabs
│   │   └── wizard.go               # Interactive component setup interview
│   └── cli/                        # Cobra + Viper CLI command suite
├── manifests/                      # Declarative YAML manifests (embedded + overlay)
│   ├── packages/                   # 80 packages categorized (core, terminal, editor, etc.)
│   ├── configs.yaml                # 23 dotfile configuration assets
│   └── profiles.yaml               # 8 workstation profiles (minimal, dev, full, etc.)
├── hacks/                          # Declarative workstation & hardware hacks
│   ├── dell-precision-m6800/       # Dell M6800 fan/power/GPU tweaks (portable $HOME)
│   └── x11-scroll-lock-led/        # Hardware Scroll Lock LED control
├── scripts/                        # Multi-platform bootstrap scripts
│   ├── bootstrap_anx.sh            # Unix pre-flight & Go bootstrap (GVM priority)
│   └── bootstrap_anx.ps1           # Windows pre-flight & Go bootstrap (go.dev archive)
├── install.sh                      # Universal installer entrypoint (delegates to bootstrap)
├── install.bat                     # Windows installer entrypoint (delegates to bootstrap.ps1)
└── uninstall.sh                    # Universal uninstaller & cache purger
```

## 3. Multicall Binary Architecture

`anx` is designed as a single multicall binary.
When invoked as `anx`, it launches the developer environment management tree.
When invoked as `aitools` (via symlink or binary copy), it dispatches directly to the AI productivity suite:

```bash
# Same binary, two personalities:
anx --version       # -> anx version v1.0.0
aitools --version   # -> aitools version 1.0.0
```

## 4. Multi-Platform Go Bootstrap Architecture

When installing on clean machines, `scripts/bootstrap_anx.sh` guarantees that Go `>= 1.24` is available through a 3-tier fallback hierarchy:
1. **Tier 1 (GVM Priority)**: Installs or sources GVM, compiles/fetches modern Go binary (`gvm install go1.24.0 -B && gvm use go1.24.0 --default`).
2. **Tier 2 (Personal Script Fallback)**: Runs `src/remotes/install_golang.sh -i`, downloading from official `go.dev/dl/?mode=json` with SHA-256 verification.
3. **Tier 3 (Package Manager Fallback)**: Native package manager (`brew`, `pacman`, `dnf`, `apt`, `apk`).

On Windows, `scripts/bootstrap_anx.ps1` downloads the official user-space archive from `go.dev`, verifies SHA-256, extracts to `%LOCALAPPDATA%\Programs\go`, configures user `PATH`, and compiles `anx.exe` and `aitools.exe`.
