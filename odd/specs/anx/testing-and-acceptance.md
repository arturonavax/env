# anx — Testing & Acceptance Criteria

## 1. Acceptance Criteria Verification

Every acceptance scenario defined in `openspec/tasks.md` maps to an automated or documented verification gate:

- **AC-001 (Native Windows Launch)**: `anx version`, `anx --help`, and `anx status` execute in PowerShell without bash or WSL.
- **AC-002 (Linux TTY Launch)**: `anx` starts inside headless console/SSH and adapts layout without Wayland/X11.
- **AC-003 (Termux Launch)**: `anx status` detects Android Termux environment and operates within prefix boundaries.
- **AC-004 (Headless Mode)**: `anx status --format json` produces valid JSON with 0 ANSI color codes and 0 interactive prompts.
- **AC-005 (Responsive Dashboard)**: Bubbletea model responds to `tea.WindowSizeMsg` resizing between 60, 100, and 160 columns without crash or panic.
- **AC-006 (Theme Selection)**: Tokyo Night active by default; theme changes persist to state file.
- **AC-007 (Package Inventory)**: Reports real installed status, detected version, and health of managed tools.
- **AC-008 (Safe Repeated Install)**: Re-running `anx apply` on an existing clean state results in 0 mutations.
- **AC-009 (Three-Way Merge)**: Detects simultaneous edits on base/local/incoming and applies non-overlapping changes.
- **AC-010 (Crash Recovery)**: Simulating an interrupted transaction leaves an incomplete journal entry, which `anx resume` detects and handles.
- **AC-011 (Safe Uninstall)**: Unmanaged tools and user dotfiles not tracked in state are preserved.
- **AC-012 (Update Semantics)**: `anx update` refreshes cached indexes without mutating installed binaries or dotfiles.
- **AC-013 (Upgrade Semantics)**: `anx upgrade` plans and checks versions before mutating.
- **AC-014 (Sync Semantics)**: `anx sync --dry-run` shows planned file operations without modifying disk.
- **AC-015 (Self-Update)**: Validates binary checksum and uses atomic swap.
- **AC-016 (Windows Executable Replacement)**: Uses rename staging to avoid locked file errors.
- **AC-017 (Security Audit)**: `anx audit` queries OSV and maps CVEs to package versions.
- **AC-018 (Unsupported Resource)**: Flags unavailable tools gracefully on unsupported platforms.
- **AC-019 (Bounded Concurrency)**: Parallel task runner uses semaphore worker pool (max jobs).
- **AC-020 (Legacy Behavior Parity)**: Dotfiles in `files/` match destinations deployed by legacy `install.sh`.
- **AC-021 (Privacy)**: Zero network calls during local commands; zero telemetry.
- **AC-022 (Low-Memory Operation)**: Memory usage strictly stays under 25MB during dashboard idle.
- **AC-023 (State Corruption)**: Corrupted JSON state file triggers graceful recovery or backup restore prompt.
- **AC-024 (Changelog Accuracy)**: Displays actual commit and tag history without fictitious entries.
