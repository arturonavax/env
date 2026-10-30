# anx — Migration Plan & Legacy Compatibility

## 1. Migration Strategy

The migration from Bash scripts to `anx` is non-destructive:
1. **Preserve Assets**: All configuration files in `files/` (`nvim`, `ghostty`, `herdr`, `atuin`, `zsh`, `starship`, `rcmd`) remain canonical and managed.
2. **Coexistence**: `install.sh` and `src/wizard.sh` remain functional during the transition period.
3. **Bridge Commands**: A lightweight shell shim or alias (`alias env-install="anx apply"`, `alias env-sync="anx sync"`) allows users to adopt `anx` incrementally.
4. **Deprecation**: Legacy Bash entry points will eventually delegate directly to `anx` once parity is validated across all supported platforms.

## 2. Command Mapping Table

| Legacy Bash Workflow | New Go `anx` Command | Notes |
|---|---|---|
| `./install.sh --wizard` | `anx wizard` | Rich interactive terminal wizard |
| `./install.sh -y` | `anx apply --yes` | Autonomous unattended install |
| `./install.sh --plan` | `anx plan` | Dry-run plan output |
| `./utils/sync_config.sh` | `anx sync` | 3-way merge dotfile synchronization |
| `./utils/backup_config.sh` | `anx backup create` | Snapshot with 50MB rotation quota |
| `./utils/restore_backup.sh` | `anx restore` | Interactive or ID-based snapshot restore |
| `./uninstall.sh --plan` | `anx uninstall --plan` | Ownership-aware uninstallation plan |
| N/A | `anx dashboard` | Tokyo Night full-screen interactive TUI |
| N/A | `anx audit` | OSV vulnerability security scanner |
| N/A | `anx doctor` | Cross-platform health checks & diagnostics |
| N/A | `anx update` | Catalog & metadata refresh without mutating |
| N/A | `anx upgrade` | Selective toolchain & package upgrade |
| N/A | `anx resume` | Interrupted transaction recovery |
