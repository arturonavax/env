# anx — Formal Requirements Specification

This document details the functional and non-functional requirements categorized by stable identifiers.

## 1. Architecture Requirements (ARCH)

- **ARCH-001**: Implement Hexagonal Architecture (Ports and Adapters). Core domain must have zero dependencies on CLI (Cobra) or TUI (Bubbletea).
- **ARCH-002**: Strict separation of concerns: `domain`, `app` (use cases), `adapters` (filesystem, platform, package providers), and `presentation` (cli, tui).
- **ARCH-003**: Domain entities must be immutable or state-transition validated.
- **ARCH-004**: Dependency injection via functional options or constructor parameters; no mutable package-level globals.

## 2. Platform Compatibility Requirements (PLAT)

- **PLAT-001**: Support Linux (Debian/Ubuntu, Arch, Fedora, openSUSE, Alpine).
- **PLAT-002**: Support macOS (Darwin amd64 and arm64 / Apple Silicon).
- **PLAT-003**: Support native Windows 10/11 (amd64 and arm64) without requiring WSL or Cygwin/MSYS.
- **PLAT-004**: Support Windows WSL2 (running Linux core).
- **PLAT-005**: Support Android Termux (ARM64) with native `pkg` provider.
- **PLAT-006**: Headless & SSH detection: adapt output when stdout is not a TTY or inside restricted terminals.

## 3. CLI Command Contract (CLI)

- **CLI-001**: Root command `anx` with subcommands: `dashboard`, `wizard`, `init`, `install`, `uninstall`, `plan`, `apply`, `status`, `list`, `search`, `info`, `doctor`, `diff`, `sync`, `update`, `upgrade`, `audit`, `backup`, `snapshot`, `restore`, `rollback`, `history`, `resume`, `profile`, `config`, `theme`, `self`, `changelog`, `completion`, `version`.
- **CLI-002**: Unified global flags: `--interactive`, `--non-interactive`, `-y`/`--yes`, `--dry-run`, `--format [text|json]`, `--quiet`, `--verbose`, `--no-color`, `--theme`, `--offline`.
- **CLI-003**: Machine-readable JSON output on `--format json` with zero ANSI escape codes.
- **CLI-004**: Predictable exit codes: 0 (success), 1 (general error), 2 (misuse/validation), 3 (conflict), 4 (interrupted), 5 (rollback required).

## 4. Terminal User Interface (TUI)

- **TUI-001**: Bubble Tea v2 Model-View-Update architecture adhering to the 4 Golden Rules (border height subtraction, explicit truncation, weight-based sizing, layout-matched mouse coordinates).
- **TUI-002**: Tokyo Night theme tokens applied consistently across background, borders, text, accents, warnings, and errors.
- **TUI-003**: Responsive layouts adapting dynamically to Wide (multi-pane), Medium (tabbed/reduced), and Narrow (single-column) terminal dimensions.
- **TUI-004**: Keyboard-first navigation (Vim keys `hjkl`, Tab/Shift-Tab, Esc, Enter, `/` for search, `?` for contextual help).

## 5. Package & Tool Management (PKG)

- **PKG-001**: Declarative catalog supporting providers: `apt`, `dnf`, `pacman`, `zypper`, `brew`, `pkg` (termux), `binary` (direct GitHub/release download), `go`, `cargo`, `git`.
- **PKG-002**: Multi-tier fallback strategies per tool (e.g. static binary first, then native package manager, then source compile).
- **PKG-003**: Version detection and comparison using Semantic Versioning.
- **PKG-004**: Ownership tracking: `anx` only uninstalls resources that it explicitly installed or marked managed.

## 6. Configuration & Synchronization (CFG)

- **CFG-001**: Three-way merge engine: Base (last synced revision), Local (current filesystem target), Incoming (repository dotfile).
- **CFG-002**: Preserve `# >>> arturonavax/env >>>` marker blocks in shell startup files (`~/.zshrc`, `~/.bashrc`).
- **CFG-003**: Conflict classification: LocalModified, IncomingModified, BothModifiedCompatible, Conflicted. Interactive or plan-driven resolution.
- **CFG-004**: Safe copy-based or symlink deployment with automatic backup before mutation.

## 7. Execution, Transactions & Recovery (TXN)

- **TXN-001**: Atomic execution plans generated before any filesystem or system mutation.
- **TXN-002**: Durable append-only transaction journal (`~/.local/state/anx/journal.jsonl`).
- **TXN-003**: Crash recovery and `anx resume` support to detect interrupted transactions and resume or compensate.
- **TXN-004**: LIFO rollback engine reversing actions when a transaction fails mid-flight.

## 8. Lifecycle & Self-Update (LIFE)

- **LIFE-001**: `anx update` refreshes cached metadata and catalogs without mutating software or dotfiles.
- **LIFE-002**: `anx upgrade` safely upgrades installed packages according to version constraints.
- **LIFE-003**: `anx sync` reconciles dotfiles with 3-way merge and conflict handling.
- **LIFE-004**: `anx self update` verifies release checksums and replaces the binary safely (using staging and rename on Windows/Unix).

## 9. Security & Auditing (SEC)

- **SEC-001**: Safe process execution: explicit argument slices, no shell string concatenation, strict timeouts, environment isolation.
- **SEC-002**: Path traversal protection: canonicalize paths and ensure target files remain within authorized directories.
- **SEC-003**: `anx audit` vulnerability scanning against OSV / GitHub Security Advisories for managed tools and versions.
- **SEC-004**: Sensitive data redaction: passwords and tokens masked in logs and snapshots.

## 10. Performance & Quality (PERF & QA)

- **PERF-001**: Startup latency: CLI < 15ms, TUI < 30ms.
- **PERF-002**: Idle resource footprint: < 25MB resident memory, 0% CPU.
- **QA-001**: Complete unit test coverage for domain logic, planning, 3-way merge, and transaction journaling.
- **QA-002**: Integration tests with hermetic/sandboxed filesystems.
