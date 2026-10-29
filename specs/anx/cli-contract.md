# anx — CLI Command Contract & Exit Codes

## 1. Global Flags

| Flag | Type | Description |
|---|---|---|
| `-i, --interactive` | bool | Force interactive prompts / wizard |
| `-y, --yes` | bool | Automatically approve safe operations |
| `--dry-run` | bool | Calculate and output plan without executing mutations |
| `--format` | string | Output format: `text` (default) or `json` |
| `-q, --quiet` | bool | Suppress non-essential output |
| `-v, --verbose` | bool | Enable detailed output |
| `--debug` | bool | Enable debug tracing and stack traces |
| `--no-color` | bool | Disable ANSI colors and styling |
| `--theme` | string | Override active theme (e.g. `tokyo-night`) |
| `--offline` | bool | Disallow network requests, use local cache only |

## 2. Command Hierarchy

- `anx` (default: launches `dashboard` in interactive TTY, prints status/help otherwise)
- `anx dashboard` — Open full-screen Tokyo Night Bubble Tea dashboard
- `anx wizard` — Upfront interactive configuration & package installer wizard
- `anx init` — Initialize anx local state, catalog, and default configuration
- `anx plan [components...]` — Generate and display execution plan for desired state
- `anx apply [plan-file]` — Execute a generated plan
- `anx install <package...>` — Install specified package(s) or components
- `anx uninstall <package...>` — Safely remove managed package(s)
- `anx status` — Inspect current environment, packages, and sync drift
- `anx list` — List managed packages, configurations, and profiles
- `anx search <query>` — Search available packages and tools
- `anx info <resource>` — Detailed metadata for a tool or configuration
- `anx doctor` — Run comprehensive environment diagnostics and health checks
- `anx diff [target]` — View 3-way differences between local dotfiles and repository
- `anx sync [target]` — Reconcile dotfiles (supports `--direction=pull|push`)
- `anx update` — Refresh package indexes and catalog metadata (no installation)
- `anx upgrade [package...]` — Upgrade installed packages to desired or latest versions
- `anx audit` — Scan managed tools for known CVEs and security advisories (OSV)
- `anx backup create` — Snapshot current configuration to backup directory
- `anx backup list` — List available backups with size and timestamp
- `anx restore [snapshot-id]` — Restore files from a specific snapshot
- `anx rollback [transaction-id]` — Reverse actions from a recorded transaction
- `anx history` — Display recent transactions and their outcome
- `anx resume` — Resume an interrupted transaction
- `anx profile [list|use|show]` — Manage named configuration profiles
- `anx theme [list|show|set]` — List, inspect, or set active theme
- `anx self update` — Securely update the anx binary itself
- `anx changelog` — View repository and application release changelog
- `anx completion <bash|zsh|fish|powershell>` — Generate shell autocompletion script
- `anx version` — Print version, commit, build date, and platform info

## 3. Exit Codes

- `0` (`ExitSuccess`): Operation succeeded completely.
- `1` (`ExitGeneralError`): Unhandled error or runtime failure.
- `2` (`ExitUsageError`): Invalid flags, missing required arguments, or syntax error.
- `3` (`ExitConflict`): Configuration conflict or dependency deadlock requiring intervention.
- `4` (`ExitInterrupted`): Process cancelled by user (SIGINT/SIGTERM) with transaction recorded.
- `5` (`ExitRollbackTriggered`): Mutation failed and compensating rollback was executed.
