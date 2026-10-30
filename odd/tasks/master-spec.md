# MASTER IMPLEMENTATION SPEC — `anx`

## Spec-Driven Development · Complete Go CLI/TUI Refactor

**Repository:** https://github.com/arturonavax/env  
**New command:** `anx`  
**Language:** Go  
**Architecture:** Modular Monolith · Hexagonal Architecture · Declarative Desired State · Transactional Execution Engine  
**Development methodology:** Spec-Driven Development (SDD) + Requirements Traceability + Test-Driven Acceptance  
**Target:** Production-grade, high-performance, cross-platform developer environment and dotfiles manager

---

# 0. YOUR ROLE AND MISSION

Act as a **Principal Go Engineer, Staff Platform Architect, Terminal UI Engineer, DevEx Specialist, and Software Security Architect**.

Your mission is to perform a comprehensive, production-quality refactor of the existing `arturonavax/env` repository, evolving it from its current predominantly Bash-based installation and configuration framework into **`anx`**, a unified, high-performance, cross-platform CLI and TUI written primarily in Go.

This is not a greenfield rewrite that discards existing work. It is an architectural migration that must **preserve, consolidate, strengthen, and extend the existing features and assets**.

The ultimate product must offer:

1. An extremely interactive, visually polished terminal dashboard.
2. A powerful, predictable, fully scriptable CLI.
3. An intelligent, declarative installation and configuration wizard.
4. A resilient, concurrent, recoverable execution engine.
5. Safe bidirectional dotfile management and intelligent three-way merges.
6. Backups, snapshots, transaction history, rollback, and recovery.
7. Package and tool discovery, installation, upgrades, and health monitoring.
8. Secure self-updates and independent update/upgrade/sync workflows.
9. Security auditing, vulnerability intelligence, and guided remediation.
10. Strong, genuine support for Linux, macOS, native Windows, WSL, and Android/Termux.
11. Excellent startup time, memory efficiency, maintainability, extensibility, and long-term developer experience.

**Your responsibility is to deliver working code and verified behavior, not merely recommendations, mockups, architectural diagrams, or documentation.**

Do not claim a capability is complete unless it has an implementation and appropriate tests.

---

# 1. METHODOLOGY: STRICT SPEC-DRIVEN DEVELOPMENT

Use **Spec-Driven Development** as the governing methodology.

Do not implement this large refactor as an unstructured sequence of code changes.

Before modifying the operational code:

1. Inspect the repository and its complete existing behavior.
2. Extract and classify requirements.
3. Create an architecture decision record for major design decisions.
4. Define explicit observable acceptance criteria.
5. Build a traceability matrix connecting requirements, implementation, and tests.
6. Define safe migration milestones.
7. Implement vertical slices that deliver independently testable functionality.
8. Validate each slice before expanding scope.
9. Preserve backward compatibility where practical.
10. Record unresolved limitations without disguising them as completed features.

Create and maintain a specification structure equivalent to:

- `specs/anx/overview.md`
- `specs/anx/requirements.md`
- `specs/anx/architecture.md`
- `specs/anx/cli-contract.md`
- `specs/anx/tui-design.md`
- `specs/anx/manifest-schema.md`
- `specs/anx/state-and-transactions.md`
- `specs/anx/synchronization.md`
- `specs/anx/security.md`
- `specs/anx/platform-compatibility.md`
- `specs/anx/testing-and-acceptance.md`
- `specs/anx/migration-plan.md`
- `specs/anx/traceability.md`

The exact file organization may be improved if justified.

Every significant requirement must have a stable ID, a priority, an owner/module, an acceptance criterion, and associated tests.

Use these requirement groups as the initial specification:

- `ARCH`: architecture and engineering constraints.
- `PLAT`: platform compatibility.
- `CLI`: command-line functionality.
- `TUI`: terminal user interface.
- `PKG`: package and tool management.
- `CFG`: configuration and dotfile management.
- `TXN`: execution, transactions, and recovery.
- `LIFE`: application lifecycle and updates.
- `SEC`: security and vulnerability management.
- `PERF`: performance.
- `QA`: testing and quality.
- `MIG`: migration and compatibility.

Preserve requirement IDs throughout the project.

Do not stop after generating the specification. **Implement the agreed architecture incrementally within the available execution context.**

---

# 2. REPOSITORY AUDIT: UNDERSTAND BEFORE REFACTORING

Start by auditing the real repository at:

https://github.com/arturonavax/env

Read `AGENTS.md` and the actual source files. Treat the repository contents as authoritative for existing behavior; do not rely solely on README descriptions.

The repository currently includes several important areas:

**Entry points:**

- `install.sh`
- `uninstall.sh`
- `src/wizard.sh`
- `Makefile`

**Existing Bash infrastructure:**

- `src/lib/platform.sh`
- `src/lib/packages.sh`
- `src/lib/backup.sh`
- `src/lib/transaction.sh`
- `src/lib/concurrency.sh`
- `src/lib/smart_append.sh`
- `src/lib/plan.sh`
- `src/lib/toolchain.sh`
- `src/lib/download.sh`
- `src/lib/wizard_engine.sh`
- `src/lib/logging.sh`
- `src/lib/colors.sh`

**Existing lifecycle utilities:**

- `utils/sync_config.sh`
- `utils/copy_config.sh`
- `utils/diff_snapshots.sh`
- `utils/backup_config.sh`
- `utils/restore_backup.sh`
- `utils/rebuild_tools.sh`
- `utils/optimize_system.sh`

**Managed assets and integrations:**

- `files/`
- `src/requirements/`
- `src/ai/`
- `src/osconfig/`
- `src/remotes/`
- `tests/`

The repository already manages or integrates tools and environments such as Zsh, Neovim/LazyVim, Ghostty, Herdr, Starship, Atuin, development toolchains, AI assistants, RTK, `aitools`, fonts, desktop integrations, VS Code, Cursor, and OS-specific configurations.

These assets must not be discarded merely because the orchestration language changes.

## Required audit deliverables

Produce an inventory identifying:

- All currently managed packages and tools.
- All supported installation strategies and fallback mechanisms.
- All managed configuration files and their destinations.
- The existing installation components and dependencies.
- The current user-facing commands and flags.
- Existing backup, sync, uninstall, and rollback semantics.
- Existing tests and their coverage.
- OS-specific operations and assumptions.
- Privileged operations.
- Known destructive or potentially unsafe operations.
- Existing implicit dependencies on Bash, GNU utilities, sudo, systemd, package managers, and interactive terminals.

**Critical migration concern:** The current `uninstall.sh` contains broad deletion rules covering software installations, configuration directories, and caches. Do not reproduce this behavior blindly in Go. Replace it with ownership-aware, plan-driven cleanup.

Build a migration compatibility matrix describing what will be **preserved, migrated, adapted, deprecated, or intentionally replaced**.

---

# 3. FUNDAMENTAL PRODUCT INVARIANTS

The following are non-negotiable.

### INV-001 — One unified application

`anx` must be one coherent Go application exposing CLI and TUI interfaces over the same domain engine.

There must not be separate installations, duplicated business logic, or inconsistent state between CLI and TUI.

### INV-002 — No mandatory runtime dependency on Bash

The core command must function on native Windows without WSL, and on supported systems without requiring Bash, Python, Node.js, or a Unix environment.

Go dependencies must be integrated at build time where possible.

Platform-specific installation adapters may invoke external package managers, shells, or native executables as appropriate, but this must never be confused with a prerequisite to start `anx` itself.

### INV-003 — Core portability is independent of package portability

`anx` must start, render its UI, inspect state, and expose commands on every target platform even when individual tools cannot be installed there.

Unsupported package operations must be reported as explicit capability limitations, not as application failures.

### INV-004 — Non-destructive by default

Never overwrite, delete, reset, or replace user data without a validated plan, ownership information, and appropriate safeguards.

### INV-005 — Deterministic operation

Equivalent manifests and state must produce equivalent plans, apart from explicitly identified environmental differences.

### INV-006 — Repeatability and idempotency

Reapplying an already satisfied desired state must not repeat destructive or unnecessary mutations.

### INV-007 — CLI/TUI parity

Anything the TUI can execute must be available through the same underlying application service, accessible via the CLI or a documented command.

### INV-008 — Offline-first and private

Local operations must work without network access whenever they do not inherently require networking.

No telemetry, external accounts, AI subscription, or hosted service may be mandatory.

### INV-009 — Honest recovery semantics

Provide transaction recovery and compensating actions, but never falsely claim that an external package manager operation or an arbitrary system modification is perfectly reversible.

### INV-010 — Measured performance

Performance requirements must be validated with profiling and benchmarks rather than assumptions based on language or framework reputation.

---

# 4. GO TECHNOLOGY STACK

## Primary recommendation

Use the following as the starting architecture:

- **Go** for the application core.
- **Cobra** for CLI commands, parsing, help, and shell completions.
- **Bubble Tea v2** for the terminal UI and event loop.
- **Lip Gloss v2** for layouts, styles, and rendering.
- **Bubbles v2** where its ready-made components materially reduce maintenance.
- Standard Go libraries wherever practical for file handling, HTTP, concurrency, context, JSON, hashing, processes, and testing.

Consider **Huh** selectively for wizard forms only if its current version and architecture integrate reliably with Bubble Tea v2 and the wider application. Do not adopt it merely for visual convenience.

Evaluate a small number of specialist dependencies only when clearly justified for:

- Reliable terminal capability detection.
- Cross-platform advisory file locking.
- Semantic version handling.
- Three-way merge algorithms.
- Structured configuration formats.
- Secure archive handling.
- Manifest validation.
- Executable update verification.
- Snapshot metadata persistence.

Do not force these capabilities into an inappropriate single framework.

**A minimal, coherent dependency set is more important than artificially insisting on one third-party package.**

## Dependency acceptance criteria

Before pinning each major dependency:

1. Verify the latest stable compatible version.
2. Confirm the actual module import path.
3. Check supported Go versions.
4. Check relevant platform limitations.
5. Examine maintainer activity and release maturity.
6. Review dependency size, licensing, and security.
7. Confirm compatibility with the selected Bubble Tea generation.
8. Document the selection in an ADR.
9. Pin versions and commit the lock metadata.
10. Run a reproducible build and automated dependency checks.

Do not mix incompatible major versions of Bubble Tea, Lip Gloss, and Bubbles.

Do not use deprecated APIs when maintained alternatives exist.

Avoid native GUI bindings, Electron, Node-based TUI runtimes, or CGO dependencies in the cross-platform core unless absolutely necessary.

---

# 5. ARCHITECTURE: MODULAR MONOLITH WITH HEXAGONAL BOUNDARIES

Implement a modular monolith with explicit separation between domain logic, application orchestration, adapters, and presentation.

An illustrative structure:

- `cmd/anx/` — executable entry point.
- `internal/cli/` — Cobra command definitions.
- `internal/tui/` — Bubble Tea models and UI components.
- `internal/theme/` — theme tokens and rendering policies.
- `internal/app/` — application services and use cases.
- `internal/domain/` — entities, rules, and typed contracts.
- `internal/manifest/` — declarative manifests and validation.
- `internal/catalog/` — package and configuration registry.
- `internal/platform/` — OS, architecture, environment, and capability discovery.
- `internal/providers/` — package manager, source, and binary adapters.
- `internal/execution/` — planning and scheduling.
- `internal/transaction/` — journals, checkpoints, and compensation.
- `internal/configsync/` — file synchronization and merge resolution.
- `internal/snapshot/` — backups, snapshots, and restoration.
- `internal/security/` — auditing and vulnerability intelligence.
- `internal/lifecycle/` — update, upgrade, and self-update orchestration.
- `internal/state/` — persistent local state.
- `internal/events/` — progress and domain event delivery.
- `internal/release/` — version and changelog metadata.
- `internal/extensions/` — validated recipes and extension mechanisms.
- `internal/testkit/` — reusable fakes, fixtures, and test adapters.

Treat this as a proposal, not a requirement to create dozens of premature packages.

**Prefer cohesive modules with narrow interfaces over excessive abstraction or a sprawling microservice-style architecture.**

## Required domain contracts

Design strongly typed interfaces for the following concepts:

- `Platform`
- `PlatformCapabilities`
- `ManagedResource`
- `Package`
- `PackageProvider`
- `InstallStrategy`
- `ConfigurationResource`
- `DesiredState`
- `ObservedState`
- `ExecutionPlan`
- `PlanStep`
- `DependencyGraph`
- `Transaction`
- `Journal`
- `Backup`
- `Snapshot`
- `Profile`
- `MergeConflict`
- `AuditFinding`
- `UpgradeCandidate`
- `Release`
- `ProgressEvent`

All presentation layers must consume application services rather than access package managers or the filesystem directly.

## Architectural boundaries

- The domain must not import Bubble Tea or Cobra.
- Package providers must not directly render terminal UI.
- Transaction orchestration must not depend on global mutable CLI state.
- The TUI must receive typed progress events.
- Long-running operations must support context cancellation.
- External processes must use explicit arguments and safe execution APIs.
- Never construct shell commands through unsafe string concatenation.
- Use structured errors with actionable context.
- Persist durable state independently of terminal presentation.

---

# 6. PLATFORM COMPATIBILITY — HIGH PRIORITY

The `anx` executable itself must support:

| Platform | Required capability |
|---|---|
| Linux distributions supported by Go | Full CLI, interactive TUI, headless mode |
| Linux without desktop environment | Full CLI and TUI through TTY or SSH |
| Linux under X11 or Wayland | Identical core behavior; no GUI dependency |
| macOS Intel and Apple Silicon | Full CLI and TUI |
| Windows 10/11 native | Full CLI and TUI without WSL |
| Windows under WSL2 | Linux execution path |
| Android through Termux | CLI and TUI, validated on actual Termux |

## Essential portability details

- Detect OS, CPU architecture, terminal capabilities, environment, and execution context separately.
- Do not infer package-management capabilities solely from `GOOS`.
- Recognize WSL without misclassifying it as native Windows.
- Recognize Termux as an Android environment with distinct filesystem conventions and native package management.
- Use platform-specific code behind narrow adapters where needed.
- Use `filepath`, `os`, `os/user` where suitable, `os/exec`, and native Go filesystem APIs.
- Use correct Windows paths, environment-variable handling, and executable suffixes.
- Do not assume POSIX signals, Unix permissions, or Unix symbolic-link privileges exist on Windows.
- Do not assume `sudo`, `/bin/sh`, `/usr/bin`, systemd, Bash, or Homebrew is available.
- On Windows, implement safe fallbacks for restricted symlinks and locked executable replacement.
- Avoid Windows console assumptions that fail in PowerShell, CMD, or Windows Terminal.
- Prefer ANSI/VT rendering only when actually supported by the terminal.
- Distinguish the host OS from a shell or compatibility layer.

## Android/Termux

Do not treat `GOOS=linux` on ARM64 as proof of native Android compatibility.

Support the appropriate Android/Termux Go build path, assess CGO requirements, and verify runtime behavior on a real Termux environment.

Do not globally force `CGO_ENABLED=0` without evaluating platform-specific consequences.

## Compatibility verification

A successful cross-compilation is not sufficient to claim full platform support.

Require actual runtime smoke tests on representative systems.

If a platform is unavailable during development, report that target as **compiled but not runtime-verified**, and provide reproducible validation instructions.

---

# 7. BOOTSTRAPPING AND DISTRIBUTION

Provide simple installation paths.

## Linux, macOS, Termux

Maintain or replace `install.sh` with a minimal, secure bootstrap entry point.

## Native Windows

Introduce `install.bat`, with PowerShell support where appropriate.

The bootstrap must:

1. Detect the target platform and architecture.
2. Choose an appropriate release binary when available.
3. Verify downloaded artifacts.
4. Install `anx` into a user-writable destination by default.
5. Avoid requiring administrative privileges merely to install the CLI.
6. Optionally install Go and build from source when explicitly requested.
7. Avoid installing package-manager or GUI dependencies unnecessarily.
8. Present actionable errors when the environment is unsupported.

**Important:** Go must be sufficient to build the application from source, but must not be required for users who install a verified precompiled executable.

Provide versioned release artifacts for the supported OS/architecture combinations.

Maintain a clearly documented support matrix distinguishing officially verified platforms from best-effort targets.

---

# 8. COMPLETE CLI DESIGN

Implement a coherent, discoverable command hierarchy.

## Top-level interface

- `anx`
- `anx dashboard`
- `anx wizard`
- `anx init`
- `anx install`
- `anx uninstall`
- `anx remove`
- `anx plan`
- `anx apply`
- `anx status`
- `anx list`
- `anx search`
- `anx info`
- `anx doctor`
- `anx check`
- `anx diff`
- `anx sync`
- `anx update`
- `anx upgrade`
- `anx audit`
- `anx backup`
- `anx snapshot`
- `anx restore`
- `anx rollback`
- `anx history`
- `anx resume`
- `anx profile`
- `anx config`
- `anx theme`
- `anx self`
- `anx changelog`
- `anx completion`
- `anx version`
- `anx help`

You may consolidate redundant aliases if the command contract remains consistent.

## Execution modes

Support:

- Fully interactive TUI.
- Interactive CLI with targeted confirmations.
- Noninteractive terminal output.
- Headless execution in CI/CD.
- Machine-readable JSON.
- Dry-run and preview.
- Offline operation where possible.
- Debug and verbose modes.
- Explicit confirmation and automation policies.

Illustrative flags:

- `--interactive`
- `--non-interactive`
- `--yes`
- `--dry-run`
- `--plan`
- `--profile`
- `--components`
- `--format`
- `--json`
- `--quiet`
- `--verbose`
- `--debug`
- `--no-color`
- `--theme`
- `--offline`
- `--jobs`
- `--timeout`
- `--backup-dir`
- `--state-dir`
- `--config`
- `--resume`
- `--rollback-on-error`
- `--force`

Document exact precedence and scope.

**`--yes` must not silently authorize unsafe operations.** Distinguish ordinary confirmation from high-risk actions requiring explicit additional consent.

## Default behavior

When invoked with no arguments:

- If an interactive TTY is suitable, launch the dashboard.
- If there is no suitable TTY, provide concise text help or a safe status summary.
- Never launch an interactive loop in CI or when stdin/stdout cannot support it.
- Never emit animation codes or ANSI color in JSON output.

Support shell completion for Bash, Zsh, Fish, and PowerShell.

Provide documented and stable exit codes.

---

# 9. TUI AND VISUAL DESIGN — TOKYO NIGHT

The TUI must be a flagship feature, not a superficial wrapper around CLI output.

Use **Tokyo Night** as the explicit default visual language.

## Required theme system

Implement a token-based theme engine supporting:

- `tokyo-night` — default.
- `tokyo-night-storm`.
- `tokyo-night-moon`, where a consistent palette can be maintained.
- `tokyo-night-day`.
- `auto`, where reliable detection exists.
- A high-contrast theme.
- A no-color mode.
- User-defined themes.

An initial Tokyo Night palette can use:

| Semantic token | Color |
|---|---|
| Background | `#1a1b26` |
| Foreground | `#c0caf5` |
| Accent blue | `#7aa2f7` |
| Cyan | `#7dcfff` |
| Purple | `#bb9af7` |
| Green | `#9ece6a` |
| Yellow | `#e0af68` |
| Red | `#f7768e` |
| Muted | `#565f89` |

Verify the complete palette against the selected Tokyo Night variants.

Do not scatter hexadecimal colors across UI components. All visual components must consume semantic tokens.

Provide:

- `anx theme list`
- `anx theme show`
- `anx theme set <name>`
- Live theme selection inside the TUI.
- Persistent theme preferences.
- Terminal-capability-aware rendering.

**The CLI must also use Tokyo Night semantics** for supported colored help, status messages, warnings, errors, and progress indicators.

Respect `NO_COLOR`, non-TTY output, user preferences, and accessibility constraints.

Do not require Nerd Fonts to display essential information.

---

# 10. FLAGSHIP INTERACTIVE DASHBOARD

Create a polished, high-density, responsive, keyboard-first dashboard.

The experience should resemble a high-quality modern terminal application, drawing inspiration from mature developer tools, interactive installers, system monitors, package managers, and projects such as gentle-shell, while maintaining its own design identity.

## Dashboard overview

The main screen must communicate the state of the development environment at a glance.

Include the following functional areas.

### A. System and environment summary

Display:

- Host OS and distribution where relevant.
- Architecture.
- Shell and terminal information when detectable.
- Current active profile.
- Installed `anx` version.
- Latest known `anx` release.
- Last synchronization time.
- Last successful transaction.
- Last backup/snapshot.
- Available updates.
- Important warnings and pending actions.

### B. Package and tool inventory

Provide a searchable, filterable, sortable inventory of all managed tools.

Each tool should expose, where known:

- Name and category.
- Installation status.
- Installed version.
- Desired version or constraint.
- Latest available version.
- Selected installation strategy.
- Source/provider.
- Supported platforms.
- Update availability.
- Health status.
- Dependency status.
- Last operation and result.
- Security advisories.
- Whether the tool was originally installed by `anx` or merely discovered.

Statuses should distinguish at least:

- Installed and healthy.
- Missing.
- Outdated.
- Update available.
- Installation in progress.
- Failed.
- Unsupported on this platform.
- Unmanaged/external.
- Unknown or not yet inspected.

Do not infer that `anx` owns a package just because the executable exists.

### C. Configuration and dotfile status

Display:

- All tracked managed configuration files.
- Expected destination paths.
- Whether each target exists.
- Managed versus untracked state.
- Clean/in-sync versus modified/drifted state.
- Local changes.
- Upstream changes.
- Merge conflicts.
- Missing files.
- Broken symlinks.
- Ownership information.
- Last sync and backup metadata.

Allow a user to select a configuration and:

- Inspect its status.
- Preview changes.
- View a diff.
- Compare local, base, and upstream revisions.
- Perform safe synchronization.
- Resolve conflicts.
- Restore a snapshot.
- Exclude it from future synchronization.

### D. Updates and changelog

Show:

- Latest `anx` release.
- Release notes and breaking changes.
- Recent repository changes relevant to managed assets.
- Available tool upgrades.
- Known tool changelogs when providers expose them.
- Last applied updates.
- Failed or deferred updates.
- Pending migration requirements.

Clearly identify the source and timestamp of each item.

Do not pretend that every package manager exposes upstream release notes.

### E. Operations and transaction history

Expose:

- Current running operations.
- Completed operations.
- Failed tasks.
- Paused/resumable jobs.
- Recent execution plans.
- Backups.
- Snapshots.
- Rollback availability.
- Logs and error details.

### F. Security overview

Expose:

- Vulnerability findings.
- Severity and confidence.
- Affected managed resources.
- Known fix versions.
- Recommended actions.
- Last scan time.
- Data sources used.
- Unsupported or unverifiable checks.

### G. Quick actions

Offer keyboard-accessible actions for:

- Install.
- Sync.
- Update.
- Upgrade.
- Self-update.
- Audit.
- Backup.
- Snapshot.
- Restore.
- Doctor.
- View changelog.
- Switch profile.
- Change theme.

## Interaction requirements

Support:

- Sidebar or tab-based navigation.
- Keyboard-first controls.
- Predictable focus management.
- Vim-style navigation where appropriate.
- Search and fuzzy filtering.
- Multi-selection.
- Contextual actions.
- Confirmation dialogs.
- Scrollable details.
- Expandable log views.
- Live progress.
- Actionable error screens.
- Contextual help.
- Clear empty states.
- Optional mouse interactions when supported.

Never let decorative features reduce usability.

---

# 11. RESPONSIVE TERMINAL RENDERING

The dashboard must adapt to:

- Large terminals.
- Ordinary laptop terminals.
- Small split panes.
- SSH sessions.
- TTY consoles.
- Windows terminals.
- Narrow mobile Termux windows.

Implement explicit responsive layouts.

For example:

- Wide: multi-panel dashboard.
- Medium: reduced panels with contextual navigation.
- Narrow: single-column layout.
- Extremely narrow: essential information only, without broken rendering.

Use Bubble Tea resize events and measured content dimensions.

Avoid arbitrary fixed widths.

Prevent:

- Horizontal overflow.
- Cut-off controls.
- Layout corruption after resizing.
- Full-screen flickering.
- Excessive redraws.
- Lost focus after updates.
- Broken Unicode rendering.

Account for grapheme clusters, wide characters, and terminal capability differences.

Provide sensible fallback behavior for terminals with limited color or Unicode support.

Terminal animations must be optional and must not waste CPU when the UI is idle.

---

# 12. INTELLIGENT INSTALLATION WIZARD

Replace the current Bash wizard experience with a rich Go implementation while preserving existing installation behavior.

## Core wizard requirements

The wizard must:

1. Inspect host capabilities.
2. Identify available installation methods.
3. Load the active profile and previous choices.
4. Discover already installed tools.
5. Show the complete component catalog.
6. Let users select packages, groups, and profiles.
7. Show dependencies and conflicts.
8. Offer applicable installation strategies.
9. Collect relevant configuration preferences.
10. Validate selections and paths.
11. Generate an execution plan.
12. Display all meaningful side effects.
13. Explain backup and rollback behavior.
14. Request appropriate approvals.
15. Execute tasks with live progress.
16. Record results and recovery metadata.
17. Display a final summary with failures and next steps.

Prefer a **complete upfront interview** before mutation, preserving the strong behavior already present in the repository.

## Installation strategies

Allow a resource to declare multiple strategies, including:

- Native package manager.
- Official release binary.
- Version manager.
- Source compilation.
- Official installer.
- Repository-managed integration.
- Custom trusted provider.

Support platform-specific package name aliases and alternatives.

Examples include differing names across `apt`, `dnf`, `pacman`, `zypper`, Homebrew, Windows package managers, and Termux `pkg`.

Implement ordered fallback policies, but do not automatically fall back to insecure installation sources.

The wizard must distinguish unavailable, unsupported, incompatible, and failed strategies.

## Safe retry behavior

On failure:

- Show the specific failing step.
- Explain its effects.
- Offer safe retry.
- Offer an alternative verified strategy where available.
- Offer resume or rollback if supported.
- Preserve the transaction log.
- Avoid blindly repeating already committed work.

---

# 13. DECLARATIVE PACKAGE AND TOOL REGISTRY

Replace hardcoded Bash package translation logic with a validated declarative registry.

Each managed resource should support appropriate metadata such as:

- Stable resource ID.
- Display name.
- Category.
- Description.
- Supported platforms and environments.
- Package aliases by provider.
- Installation providers.
- Strategy priority.
- Version detection.
- Desired version or version range.
- Dependencies.
- Conflicts.
- Installation prerequisites.
- Privilege requirements.
- Health checks.
- Update checks.
- Upgrade strategy.
- Uninstall/ownership strategy.
- Configuration resources.
- Estimated side effects.
- Rollback and compensation capabilities.
- Security ecosystem identifiers.
- Documentation links.
- Source integrity metadata.

Use a schema-versioned configuration format with validation, examples, and migrations.

Prefer a manifest model that is expressive enough for real workflows but does not become an unrestricted scripting language.

## Resource capabilities

A provider must independently declare whether it can:

- Detect installation.
- Detect version.
- Install.
- Upgrade.
- Remove safely.
- Pin versions.
- Query latest versions.
- Verify health.
- Report vulnerabilities.
- Provide rollback information.

Do not assume every provider supports all capabilities.

## Extensibility

Enable future user-defined packages, profiles, and validated recipes without modifying the Go core.

Avoid runtime Go plugins or unsafe dynamic code loading as the default extension mechanism.

If arbitrary scripts are supported, require explicit trust, inspectable execution plans, permissions disclosure, and documented limitations.

---

# 14. PLANNING, DEPENDENCY GRAPHS, AND CONCURRENCY

The execution engine must operate from a typed, inspectable plan.

## Plan stages

1. Discover observed state.
2. Resolve desired state.
3. Validate platform capabilities.
4. Resolve dependencies and conflicts.
5. Select providers.
6. Calculate required changes.
7. Identify privileged and destructive operations.
8. Identify files and resources affected.
9. Establish task ordering.
10. Determine safe parallelism.
11. Produce a previewable immutable plan.
12. Execute with progress and journaling.

Use a directed acyclic graph where appropriate.

Detect and explain dependency cycles.

## Concurrency

Implement bounded concurrency with:

- Configurable worker limits.
- Backpressure.
- Context cancellation.
- Resource-level locks.
- Package-manager locks.
- Filesystem mutation locks.
- Network concurrency limits.
- Shared-cache coordination.
- Explicit task dependency tracking.

Run independent tasks concurrently where safe.

Do not run conflicting package-manager operations in parallel.

Do not allow concurrent writes to the same destination.

Do not launch an unbounded number of goroutines.

Expose task progress through structured events consumed by CLI and TUI renderers.

Make concurrency deterministic and testable.

---

# 15. TRANSACTIONS, JOURNALS, RESUME, AND ROLLBACK

Build a durable transaction engine that improves on the current Bash implementation.

## Required behavior

Every mutating operation must use the same underlying execution framework.

This includes installation, configuration synchronization, self-updates, removal, restoration, and eligible upgrades.

Persist:

- Transaction ID.
- Plan ID and checksum.
- Start/end timestamps.
- Relevant platform and profile information.
- Resource identifiers.
- Planned actions.
- Preconditions.
- Original state metadata.
- Snapshots and backup references.
- Step status.
- Completed actions.
- Errors.
- Compensation eligibility.
- Recovery state.

## Transaction lifecycle

Support explicit states such as:

- Planned.
- Validated.
- Awaiting approval.
- Running.
- Interrupted.
- Failed.
- Partially applied.
- Committed.
- Rolled back.
- Partially rolled back.
- Recovery required.

## Crash consistency

Implement durable journaling with appropriate ordering:

- Record intent and required recovery information before mutations.
- Write temporary files safely.
- Use atomic replacement where supported.
- Flush durable metadata where needed.
- Verify preconditions before applying changes.
- Recover from interrupted processes.
- Detect and avoid conflicting concurrent application instances.

## Rollback

Rollback must:

- Restore owned configuration changes when the original state is available.
- Restore metadata where supported.
- Reverse actions in dependency-safe order.
- Preserve user modifications made after a transaction.
- Avoid deleting unrelated resources.
- Distinguish fully compensated and partially compensated operations.

**Never claim universal atomic rollback for external package managers.**

When package installation, operating-system changes, or third-party installers cannot be safely reversed, mark those steps explicitly and provide recovery guidance.

## Resume

Implement `anx resume` with:

- Transaction inspection.
- Checkpoint validation.
- Recovery after interruption.
- Idempotent continuation.
- Skipping already satisfied steps.
- Revalidation when the environment has changed.

Test simulated crashes and cancellation at mutation boundaries.

---

# 16. SAFE DOTFILE SYNCHRONIZATION AND THREE-WAY MERGING

This is a core feature.

Preserve the repository's current non-destructive configuration philosophy, including `smart-append-first` and managed shell marker blocks.

Extend it to a robust synchronization engine.

## Three-way merge model

Track:

- **Base:** last successfully applied managed revision.
- **Local:** current destination state.
- **Incoming:** new desired managed revision.

Classify outcomes such as:

- No change.
- Incoming-only change.
- Local-only change.
- Nonconflicting simultaneous changes.
- Conflicting simultaneous changes.
- Deleted locally.
- Deleted upstream.
- Untracked or external file.
- Invalid or inaccessible target.

Use an actual three-way merge algorithm when appropriate.

Never call a two-way text comparison a three-way merge.

## File handling

Support:

- Plain text.
- Structured configuration files where safe.
- Symlinks.
- Copy-based deployment.
- Directories.
- Executable bits and file modes where supported.
- Platform-specific path normalization.
- CRLF/LF considerations.
- Encodings and BOM handling.
- Managed marker blocks.
- Secret exclusion policies.

For binary or unsafe-to-merge files, require explicit conflict resolution.

Do not attempt semantic merges of complex formats unless a tested parser and well-defined merge policy exist.

## Conflict resolution

Provide:

- Preview.
- Keep local.
- Take incoming.
- Merge automatically where safe.
- Manually resolve.
- Skip.
- Restore prior snapshot.

Allow conflict handling from both CLI and TUI.

Do not silently choose incoming when the user has modified local content.

## Bidirectional workflows

Support separate, explicit concepts:

- Applying repository-managed configuration to the machine.
- Capturing eligible local configuration changes back into a working repository.
- Inspecting differences before transfer.
- Reviewing conflicts and exclusions.

Never automatically push Git commits or upload local secrets merely because `sync` was requested.

Design safe handling of simultaneous changes across multiple machines.

---

# 17. BACKUPS, SNAPSHOTS, RESTORATION, AND PROFILES

## Backups

Provide incremental or content-aware backup behavior when practical.

Backups must preserve enough metadata to restore eligible files correctly.

Support:

- Explicit backup creation.
- Automatic pre-mutation snapshots.
- Retention policies.
- Disk quota management.
- Snapshot listing.
- Snapshot inspection.
- Snapshot diffing.
- Verified restoration.
- Corruption detection.
- Safe garbage collection.

Do not delete the only recoverable backup merely to satisfy a quota.

## Snapshots

A snapshot should represent a coherent inventory of:

- Managed configuration state.
- Resource versions where discoverable.
- Active profile.
- Desired-state manifest revision.
- Relevant environment metadata.
- Backup references.
- Transaction references.

Distinguish declarative state snapshots from complete machine images.

Provide meaningful comparison and restoration behavior.

## Profiles

Implement named profiles such as:

- `minimal`
- `developer`
- `workstation`
- `server`
- `ai`
- `custom`

Treat names as examples; preserve existing component taxonomy.

Support:

- Profile creation.
- Profile selection.
- Profile composition.
- Dependency resolution.
- Per-platform overrides.
- Desired package sets.
- Managed configuration sets.
- Tool version policies.
- Theme preferences.
- Sync policies.
- Export/import.
- Validation.
- Profile diffing.

Keep machine-local sensitive settings out of sharable profiles by default.

---

# 18. LIFECYCLE COMMAND SEMANTICS — UPDATE, UPGRADE, SYNC

These commands must be clearly differentiated.

Do not implement them as three aliases for the same action.

## `anx update`

**Purpose:** Refresh available knowledge and discover changes.

Responsibilities:

- Refresh supported package metadata.
- Check managed resource versions.
- Check available upstream releases.
- Fetch relevant catalog and repository metadata where configured.
- Detect a newer `anx` release.
- Refresh cached changelog information.
- Recalculate available upgrades.
- Report pending desired-state changes.

By default, `anx update` must **not** upgrade installed tools, overwrite dotfiles, or silently replace the `anx` executable.

Provide offline behavior using cached metadata when available.

## `anx upgrade`

**Purpose:** Upgrade selected installed packages, tools, or toolchains.

Responsibilities:

- Identify upgrade candidates.
- Respect version pins and profile policies.
- Select compatible providers.
- Explain dependency consequences.
- Show release notes where available.
- Generate a plan.
- Create appropriate backups.
- Execute approved upgrades.
- Validate post-upgrade health.
- Record applied versions.
- Offer recovery actions.

Support scoped upgrades:

- All eligible managed tools.
- One component.
- One package.
- One profile.
- Security fixes only, where reliable classification is available.

Do not equate package upgrades with configuration synchronization.

## `anx sync`

**Purpose:** Reconcile managed configuration and desired state.

Responsibilities:

- Compare base/local/incoming configuration revisions.
- Detect drift.
- Generate an explicit reconciliation plan.
- Apply safe configuration changes.
- Resolve conflicts.
- Record updated base revisions.
- Preserve unrelated user files.
- Produce a synchronization report.

Support distinct safe directions or policies:

- Pull managed configuration changes.
- Capture eligible local modifications.
- Review bidirectional changes.

Do not silently overwrite configurations or push to remote Git repositories.

## `anx self update`

**Purpose:** Securely update the `anx` executable itself.

Responsibilities:

- Check available application releases.
- Select the correct OS/architecture artifact.
- Inspect release metadata.
- Verify integrity and authenticity.
- Download to staging.
- Validate the candidate.
- Replace or activate the executable safely.
- Preserve user configuration and state.
- Provide recovery/rollback if supported.

Handle native Windows executable locking safely.

Never assume an executable can replace itself while running.

## Auto-update policy

Support configurable policies for:

- Checking for updates on launch.
- Displaying update notifications.
- Automatic metadata refresh.
- Explicitly opt-in unattended upgrades.
- Scheduled checks through suitable platform mechanisms.

Default behavior must prioritize user control and predictable operation.

No mandatory persistent daemon.

Unattended security-sensitive changes must use a clearly documented trust and approval model.

---

# 19. SECURITY AUDIT AND VULNERABILITY MANAGEMENT

Implement `anx audit` as a meaningful developer environment security auditor.

Do not confuse a package inventory with a vulnerability scan.

## Data sources

Evaluate appropriate integrations with:

- OSV.
- GitHub Security Advisories.
- NVD where reliable matching exists.
- Package-manager-native vulnerability metadata.
- Other trustworthy, documented sources when justified.

Do not require authentication for functionality that can operate anonymously.

Support optional credentials securely and avoid leaking them in logs.

## Audit capabilities

Audit:

- Managed package versions.
- Known vulnerable software versions.
- Outdated runtimes and toolchains.
- Configuration security issues with tested checks.
- Insecure file permissions where meaningful.
- Unsafe installation-source configuration.
- Integrity failures.
- Available security updates.
- Relevant local environment misconfigurations.

## Finding model

Each result should include:

- Affected resource.
- Detected version.
- Vulnerability identifier.
- Source.
- Severity.
- Confidence.
- Applicability.
- Affected-version evidence.
- Fixed version when known.
- Recommended remediation.
- Remediation availability.
- Scan timestamp.

Avoid inaccurate matches across unrelated package ecosystems.

If there is insufficient evidence to map a tool to an advisory, report an unknown/unverified status rather than inventing a finding.

## Remediation

Provide:

- Preview remediation.
- Apply safe, available upgrades.
- Open or display relevant advisory details.
- Skip or acknowledge a finding.
- Export reports.

Remediation must pass through the same planning, backup, confirmation, and transaction system used elsewhere.

Never automatically execute arbitrary AI-generated security scripts.

---

# 20. SYSTEM HEALTH AND DIAGNOSTICS

Implement `anx doctor` and related commands.

Inspect:

- Installation completeness.
- Missing dependencies.
- Broken managed configurations.
- Invalid manifest entries.
- Package provider failures.
- Version mismatches.
- Unhealthy tools.
- Broken symlinks.
- Unexpected state drift.
- Failed or interrupted transactions.
- Snapshot integrity.
- Available disk space.
- Relevant permission issues.
- Network capability where necessary.
- Terminal capability issues.

Provide actionable recommendations.

Distinguish:

- Verified healthy.
- Warning.
- Failure.
- Unsupported.
- Unknown.
- Not checked.

Provide both human-readable and structured JSON reports.

Do not classify an unsupported optional package as an unhealthy application.

---

# 21. EXTERNAL PROCESS AND PRIVILEGE SAFETY

Package managers, installers, and configuration scripts are high-risk integration points.

Implement:

- Safe argument-based process execution.
- Explicit working directories.
- Controlled environment variables.
- Timeouts and cancellation.
- Redacted logs.
- Exit-code capture.
- Output size limits.
- Streaming progress when useful.
- Command provenance.
- Source integrity verification.
- Privilege minimization.
- Explicit elevation prompts where appropriate.

Never automatically run arbitrary downloaded shell scripts from untrusted sources.

Do not use a broad elevated shell session merely for convenience.

Avoid persistent credentials.

Use canonicalized paths and ownership checks to prevent accidental or malicious operations outside approved destinations.

Protect against:

- Path traversal.
- Symlink attacks.
- Time-of-check/time-of-use races where practical.
- Corrupted downloads.
- Untrusted manifests.
- Dependency confusion.
- Rollback data corruption.
- Accidental secret disclosure.
- Concurrent state modification.

---

# 22. SOURCE CONTROL AND RELEASE MANAGEMENT

The GitHub repository is the source of the application and the managed configuration catalog.

However, distinguish:

1. `anx` application releases.
2. Repository configuration revisions.
3. Managed package versions.
4. Local machine state.

Do not use one global version number to describe all four.

## Release pipeline

Implement a reproducible release process with:

- Semantic application versioning.
- Structured release notes.
- `CHANGELOG.md`.
- Automated build artifacts.
- Artifact checksums.
- Release authenticity verification.
- Cross-platform archives.
- Version embedding.
- Compatibility checks.
- Rollback-aware self-update support.

Consider release signing or trusted build attestations where justified.

Do not treat a checksum hosted next to an artifact as sufficient protection against a compromised download source unless there is a trustworthy authentication mechanism.

## Changelog integration

The dashboard and `anx changelog` must distinguish:

- Application release changes.
- Configuration catalog changes.
- Managed tool upstream changes.
- Local operation history.

Show cached results offline where available.

---

# 23. PERFORMANCE AND RESOURCE EFFICIENCY

The project prioritizes excellent performance, including low-memory machines.

Optimize for:

- Fast startup.
- Low idle CPU usage.
- Low idle RAM consumption.
- Responsive keyboard input.
- Responsive resize handling.
- Minimal terminal redraw work.
- Fast local state inspection.
- Bounded concurrency.
- Efficient filesystem operations.
- Limited network requests.
- Incremental state refresh.
- Avoidance of redundant package-manager queries.

## Performance engineering requirements

Use:

- Go benchmarks.
- `pprof`.
- Allocation profiling.
- Race detection.
- Representative fixtures.
- Real terminal rendering measurements.
- Startup and memory measurement scripts.

Establish a reproducible reference environment and record a performance baseline.

Set measurable budgets for:

- CLI cold start.
- TUI cold start.
- TUI idle memory.
- Idle CPU.
- Inventory refresh time.
- Dashboard responsiveness.
- Rendering under frequent progress events.
- Large manifest processing.

Prefer explicit thresholds based on measured reference hardware rather than unsupported performance promises.

Avoid polling when event-driven updates are available.

Do not turn every lightweight status query into a full external package registry refresh.

---

# 24. STATE STORAGE AND CONFIGURATION

Design a versioned local state format.

It must support:

- Transaction journals.
- Desired state.
- Observed state cache.
- Profile selections.
- Manifest revisions.
- Configuration merge bases.
- Backup and snapshot indexes.
- Package ownership records.
- Update metadata.
- Audit history.
- User preferences.

Evaluate whether an embedded database is justified or whether versioned, atomic files are sufficient.

Document the trade-off and choose the simpler reliable approach.

## Storage principles

- Cross-platform locations.
- User-writable defaults.
- Schema versioning.
- Migration support.
- Atomic updates where possible.
- Concurrent access protection.
- Recovery from corruption.
- Explicit state reset commands.
- No secrets in ordinary logs or sharable snapshots.
- Clear separation of configuration, cache, and durable state.

Use appropriate platform conventions rather than hardcoding Unix home-directory layouts on Windows.

---

# 25. MIGRATION OF EXISTING FUNCTIONALITY

Migrate the existing Bash functionality in controlled stages.

## Existing capabilities to preserve

- Platform discovery.
- Package translation.
- Fallback installation strategies.
- Core requirements installation.
- Terminal environment installation.
- Editor installation.
- AI tooling installation.
- Font installation.
- OS integrations.
- Safe shell configuration modification.
- Backups.
- Planning and dry-run.
- Transaction history.
- Rollback.
- Concurrent task execution.
- Configuration synchronization.
- Snapshot comparison.
- Restore.
- Toolchain rebuilding.
- Diagnostic operations.
- Interactive and noninteractive modes.

## Migration principles

- Do not rewrite standalone working tools merely because they are not written in Go.
- Preserve managed configuration assets in `files/`.
- Preserve existing tool behavior unless a change is necessary and documented.
- Migrate orchestration into Go first.
- Retain narrowly scoped OS-specific scripts as provider implementations when appropriate.
- Introduce explicit compatibility adapters for legacy workflows.
- Deprecate obsolete entry points only after parity has been verified.
- Maintain documentation showing equivalent old and new commands.

Update `AGENTS.md`, `README.md`, the Makefile, and relevant developer documentation as the architecture changes.

Do not allow documentation to describe unimplemented behavior.

---

# 26. AUTOMATED TESTING STRATEGY

Testing must be designed into the architecture.

## Unit tests

Cover:

- Manifest parsing and validation.
- Dependency resolution.
- Platform capability detection.
- Package strategy selection.
- Plan generation.
- Idempotency.
- State transitions.
- Conflict classification.
- Three-way merge behavior.
- Backup metadata.
- Snapshot restoration.
- Transaction journaling.
- Rollback decisions.
- Security finding normalization.
- Theme token selection.
- CLI argument parsing.

## Integration tests

Use fake providers and sandboxed filesystems for mutating workflows.

Cover:

- Fresh installation.
- Already installed tools.
- Partial installation.
- Missing dependencies.
- Interrupted installation.
- Retried installation.
- Resume after failure.
- Rollback after interruption.
- Concurrent conflicting operations.
- Configuration drift.
- Simultaneous local and incoming changes.
- Restore after a bad sync.
- Self-update failure.
- Offline operation.
- Unsupported package on an otherwise supported OS.

## TUI tests

Cover:

- Model update logic.
- Navigation.
- Focus.
- Confirmation behavior.
- State rendering.
- Terminal resize.
- Narrow layouts.
- Error screens.
- No-color rendering.
- Theme switching.
- Cancellation.
- Progress updates.

Use deterministic model tests and suitable terminal integration tests.

## Security tests

Cover:

- Path traversal.
- Symlink race scenarios where applicable.
- Unsafe deletion boundaries.
- Malicious manifest inputs.
- Corrupt archives.
- Invalid checksums/signatures.
- Untrusted update metadata.
- Concurrent writes.
- Sensitive data redaction.
- Interrupted transactions.

Add fuzz tests for parsers, merge inputs, and other high-risk components where useful.

---

# 27. CROSS-PLATFORM CI MATRIX

Create a meaningful automated build and test matrix.

At minimum:

- Linux amd64.
- Linux arm64.
- macOS amd64.
- macOS arm64.
- Windows amd64.
- Windows arm64 where the selected dependencies and CI infrastructure allow it.
- Android/Termux arm64 build validation.
- WSL-oriented smoke tests where practical.

Also assess Linux environments such as Alpine/musl when relevant.

Required CI gates:

- `go fmt` / formatting.
- `go vet`.
- Unit tests.
- Integration tests.
- Race detector on supported runners.
- Static analysis.
- Dependency vulnerability checks.
- Reproducible release builds.
- CLI smoke tests.
- Native Windows execution smoke tests.
- Headless/JSON output validation.
- Installer validation.
- Documentation consistency checks.

**Android/Termux must be runtime-tested separately before being marked verified.**

A Go cross-build alone does not satisfy that acceptance gate.

Preserve existing Bash tests as long as Bash compatibility code remains supported.

---

# 28. ACCEPTANCE TESTS — NON-NEGOTIABLE USER STORIES

Implement and verify at least the following representative scenarios.

### AC-001 — Native Windows launch

Given Windows 11 without WSL, when the user installs the correct `anx` executable, `anx --help`, `anx version`, and `anx dashboard` must function without Bash.

### AC-002 — Linux TTY launch

Given Linux without a desktop environment, `anx` must run with an interactive terminal and adapt its interface without depending on X11 or Wayland.

### AC-003 — Termux launch

Given a supported Android/Termux environment, the CLI and TUI must start and basic state inspection must work independently of whether desktop-oriented tools are installable.

### AC-004 — Headless mode

Given a process without a usable TTY, `anx status --json` must produce valid machine-readable output with no ANSI sequences or interactive prompts.

### AC-005 — Responsive dashboard

Given a running dashboard, resizing the terminal across wide, medium, and narrow widths must not corrupt navigation or render unusable layouts.

### AC-006 — Theme selection

Given the default installation, Tokyo Night must be active. After selecting another supported theme, the preference must persist across invocations.

### AC-007 — Complete package inventory

Given managed resources, the dashboard must show detected installation status, provider, known versions, health, and available updates without inventing unknown metadata.

### AC-008 — Safe repeated installation

Given an already satisfied desired state, rerunning installation must not overwrite unrelated configuration or repeat unnecessary changes.

### AC-009 — Three-way merge

Given a managed file changed locally and upstream since the last applied base, `anx sync` must detect both sides and perform a correct merge or request conflict resolution.

### AC-010 — Crash recovery

Given a process interrupted during a multi-step mutating operation, the next invocation must detect the incomplete transaction and offer valid recovery options.

### AC-011 — Safe uninstall

Given a resource not owned by `anx`, uninstall must not remove it merely because its executable or configuration path resembles a managed resource.

### AC-012 — Update semantics

Given available new package metadata, `anx update` must refresh knowledge without automatically installing an upgrade.

### AC-013 — Upgrade semantics

Given an eligible managed tool update, `anx upgrade` must show the proposed mutation, prerequisites, and recovery limitations before applying it.

### AC-014 — Sync semantics

Given configuration drift, `anx sync --dry-run` must show the proposed reconciliation without mutating the target files.

### AC-015 — Self-update

Given an authentic compatible newer `anx` release, `anx self update` must stage, verify, and safely activate the update without losing state.

### AC-016 — Windows executable replacement

Given a Windows-native executable that cannot be overwritten while running, self-update must use a safe staged replacement strategy.

### AC-017 — Security audit

Given a verifiably vulnerable installed tool, `anx audit` must associate the advisory with the correct resource and version and provide an evidence-based remediation path.

### AC-018 — Unsupported resource

Given a package unsupported on the current platform, the dashboard must display an unsupported status while the application and other operations continue to function.

### AC-019 — Bounded concurrency

Given multiple independent and conflicting tasks, the executor must parallelize safe tasks and serialize operations sharing exclusive resources.

### AC-020 — Legacy behavior parity

Given an existing supported installation workflow, its migrated Go equivalent must preserve required behavior or document a validated, intentionally changed contract.

### AC-021 — Privacy

Given ordinary local operations, no user configuration, secrets, machine inventory, or telemetry may be transmitted without a clear feature requirement and user authorization.

### AC-022 — Low-memory operation

Given representative modest hardware, the dashboard must remain responsive and within documented, measured memory and CPU budgets.

### AC-023 — State corruption

Given corrupted or incomplete application state, `anx` must detect the issue and provide safe recovery rather than silently overwriting valid data.

### AC-024 — Changelog accuracy

Given a release with known provenance, the dashboard must display the correct source and revision information; unknown release notes must not be fabricated.

---

# 29. IMPLEMENTATION ROADMAP

Implement the migration as sequential, verifiable vertical slices.

## Phase 0 — Discovery and specification

- Audit the full repository.
- Inventory existing functionality.
- Identify risks and unsafe operations.
- Create specifications and ADRs.
- Build requirements traceability.
- Establish migration baselines.

**Exit gate:** Documented requirements and executable baseline tests.

## Phase 1 — Go foundation

- Go module.
- Cobra command structure.
- Logging and error contracts.
- Platform detection.
- Theme engine.
- Configuration and state directories.
- Cross-platform build pipeline.
- Basic CLI and TUI shell.

**Exit gate:** `anx` starts and basic commands work on validated host targets.

## Phase 2 — Declarative model and read-only dashboard

- Manifest schema.
- Package catalog.
- Profile model.
- Provider discovery.
- Version detection.
- Configuration inventory.
- Status and doctor services.
- Tokyo Night dashboard.

**Exit gate:** Real resource inventory and configuration status, without relying on hardcoded UI mock data.

## Phase 3 — Planning and execution

- Desired-state resolution.
- Dependency graph.
- Provider selection.
- Dry-run plans.
- Bounded worker execution.
- Resource locks.
- Progress event stream.

**Exit gate:** A complete safe install workflow with repeatability tests.

## Phase 4 — Durable transactions

- Persistent journals.
- Backup integration.
- Checkpoints.
- Interruption detection.
- Resume.
- Compensation and rollback.
- Ownership-aware removal.

**Exit gate:** Failure injection and recovery scenarios pass.

## Phase 5 — Intelligent configuration management

- Merge base tracking.
- Three-way merges.
- Conflict resolution.
- Bidirectional workflows.
- Snapshots.
- Profile reconciliation.

**Exit gate:** Config changes can be synchronized without destroying independent user edits.

## Phase 6 — Complete lifecycle

- Update.
- Upgrade.
- Sync.
- Self-update.
- Release metadata.
- Changelogs.
- Verified downloads.
- Windows-safe binary replacement.

**Exit gate:** Independent, correctly defined lifecycle commands work and pass safety tests.

## Phase 7 — Security intelligence

- Audit adapters.
- Vulnerability matching.
- Finding normalization.
- Remediation planning.
- Security dashboard integration.

**Exit gate:** Verified advisories and remediations work end to end.

## Phase 8 — Migration completion and hardening

- Finish migrating existing components.
- Preserve or replace legacy entry points.
- Complete documentation.
- Expand CI.
- Profile and optimize.
- Validate target platforms.
- Remove obsolete code only after parity.
- Prepare releases.

**Exit gate:** Every mandatory requirement has implementation evidence and an acceptance-test result.

---

# 30. ENGINEERING QUALITY RULES

Follow these standards throughout implementation.

## Go code quality

- Idiomatic Go.
- Small cohesive interfaces.
- Context propagation.
- Explicit dependency injection.
- Narrow package boundaries.
- Structured errors.
- Minimal global mutable state.
- No unnecessary reflection.
- No premature generic abstractions.
- No speculative complexity.
- Efficient allocations.
- No uncontrolled goroutines.
- No silent error suppression.
- No untested platform-specific assumptions.

## Application quality

- Stable command contracts.
- Explicit side effects.
- Machine-readable output.
- Predictable exit codes.
- Human-readable error guidance.
- Safe retry behavior.
- Clear progress reporting.
- No fabricated success statuses.
- Correct cancellation.
- Durable transaction history.
- User-owned configuration preservation.

## Product quality

- Consistent Tokyo Night presentation.
- Keyboard-first interaction.
- Responsive layouts.
- Accessible alternatives.
- High information density without clutter.
- Consistent terminology.
- Clear safe/unsafe operation distinction.
- Useful diagnostics.
- Comprehensive documentation.

---

# 31. THINGS YOU MUST NOT DO

- Do not delete the current application before preserving equivalent functionality.
- Do not replace a working feature with a mock implementation.
- Do not make Bash a requirement for native Windows execution.
- Do not equate Windows/WSL support with native Windows support.
- Do not treat Termux as an ordinary GNU/Linux distribution.
- Do not claim runtime compatibility based solely on successful compilation.
- Do not build a TUI tightly coupled to package-manager internals.
- Do not put configuration synchronization logic inside UI components.
- Do not implement unbounded parallel installation.
- Do not run competing package-manager mutations concurrently.
- Do not invent vulnerability findings.
- Do not automatically download and execute untrusted installation scripts.
- Do not silently overwrite local configuration changes.
- Do not recursively delete broad application directories without ownership verification.
- Do not silently erase failed transactions.
- Do not promise atomic rollback for external operations that cannot be reversed.
- Do not silently self-update from unverified artifacts.
- Do not claim an updater is safe without testing interruption and platform-specific replacement.
- Do not add unnecessary persistent daemons.
- Do not require internet access for local state inspection.
- Do not require Nerd Fonts.
- Do not add telemetry by default.
- Do not bury implementation gaps under TODO comments while reporting completion.
- Do not make undocumented breaking changes to existing workflows.
- Do not sacrifice reliability or maintainability merely to minimize the dependency count.

---

# 32. FINAL DELIVERABLES

Deliver a functioning implementation and the supporting engineering artifacts.

Required outputs:

1. A buildable Go application exposing the `anx` command.
2. A functional interactive Tokyo Night dashboard.
3. A configurable visual theme system.
4. A scriptable, headless-compatible CLI.
5. A declarative managed-resource catalog.
6. Platform-specific capability and provider adapters.
7. A validated plan/execution system.
8. Durable transaction and recovery infrastructure.
9. Ownership-aware package/configuration management.
10. Three-way configuration synchronization.
11. Backups, snapshots, restoration, and profiles.
12. Update, upgrade, sync, and verified self-update workflows.
13. Security auditing with accurate vulnerability provenance.
14. Changelog and update-history integration.
15. Secure bootstrap scripts for Unix-like systems and native Windows.
16. Cross-platform builds and tests.
17. A migration and compatibility guide.
18. Complete, accurate CLI/TUI documentation.
19. Architecture decisions and requirement traceability.
20. A final verification report.

## Final verification report

Your final implementation report must include:

- Requirements implemented.
- Requirements partially implemented.
- Requirements not implemented.
- Tests executed and their results.
- Platforms actually runtime-verified.
- Platforms only cross-compiled.
- Security limitations.
- Rollback limitations.
- Performance measurements.
- Breaking changes.
- Migration instructions.
- Remaining risks and follow-up work.

Never label the project fully production-ready if mandatory acceptance tests or platform runtime checks are missing.

---

# 33. EXECUTION INSTRUCTIONS FOR THE CODING AGENT

Begin with the current `main` branch and inspect the real codebase.

Before writing implementation code, provide a concise summary of:

1. Existing architecture and functionality.
2. Critical migration risks.
3. Architecture decisions.
4. Proposed implementation phases.
5. First vertical slice to implement.

Then create the specification files and proceed to implement.

Make incremental, reviewable changes. Preserve existing test coverage and add new tests for every meaningful subsystem.

Prioritize working end-to-end functionality over producing broad but disconnected scaffolding.

When choosing between approaches:

1. **Safety and data preservation** come first.
2. **Actual cross-platform operation** comes second.
3. **Architectural correctness and maintainability** come third.
4. **User experience and performance** must be continuously optimized, not postponed indefinitely.
5. **Dependency minimization** is valuable only when it does not compromise the previous objectives.

Use your engineering judgment on minor implementation details without interrupting progress unnecessarily.

For material architectural trade-offs, document the decision and its consequences.

Do not create fake fixtures to make an unfinished dashboard look complete.

Do not edit unrelated personal configuration files in the development machine while building or testing the project.

Use isolated test environments and explicit opt-in for integration tests requiring privileges or actual host mutations.

**Success means that `anx` becomes a genuinely reliable, beautiful, extremely capable and extensible cross-platform command-line application—not simply a Go rewrite of the existing Bash scripts.**

The target experience is a unified terminal-native control center for the user's entire managed development environment: installation, configuration, synchronization, updates, upgrades, health, security, recovery, and long-term maintenance.

**Start by auditing `arturonavax/env`, creating the traceable specification, and implementing the first production-quality vertical slice.**

