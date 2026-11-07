# ODD Task Board: System & AI Suite Comprehensive Overhaul

Tracking task board for the 13 environment, daemon, TUI, and manifest improvements.

## Tasks

### Unit 1: AI Quota & Daemon Hardening
- [x] Fix Cobra argument validation and `--help` documentation for `anx tools ai daemon auto-wake [on|off]` (`internal/ai/cli/quota_cmds.go`)
- [x] Eliminate morning re-suspend bug by distinguishing user resume from autonomous wake in D-Bus monitor (`internal/ai/daemon/daemon.go`)
- [x] Minimize wakeup probe token consumption (isolated tmpdir, `--disable-slash-commands`, `--effort low`, skip probe when window active)
- [x] Overhaul `anx tools ai quota` to display all sub-quotas (Gemini, Claude/GPT, Argon), weekly & 5h windows, with Tokyo Night dark + red accent theme (`internal/ai/quota/quota.go`, `internal/ai/quota/watch.go`, `internal/ai/cli/quota_cmds.go`)

### Unit 2: Manifests & Tool Registry
- [x] Add `tree` package to all profiles in `manifests/profiles.yaml`
- [x] Guard `~/.config/rcmd/rcmd.conf` from overwrite unless `--destroyer` (`manifests/configs.yaml`, `internal/app/sync_service.go`)
- [x] Register `rcmd` as an official personal tool and multicall binary (`manifests/tools.yaml`, `cmd/anx/main.go`)
- [x] Ensure no default browser is enforced on target systems

### Unit 3: Hacks Idempotency & Machine-Specific Config Overlays
- [x] Remove all repository file mutations from `manifests/hacks/dell-precision-m6800/apply_hacks.sh`
- [x] Make all hacks strictly idempotent (`dell-precision-m6800`, `dell-xps13-9343`, `x11-scroll-lock-led`)
- [x] Implement declarative machine-specific config overrides/diffs in `manifests/hacks.yaml` and sync engine (`internal/app/sync_service.go`, `internal/domain/resource.go`) so local overrides do not produce false drift
- [x] Display detected hacks and descriptions in the installer wizard (`internal/tui/wizard.go`)

### Unit 4: Installer TUI & Destroyer Enhancements
- [x] Implement concurrent package uninstalls and streaming log output in Destroyer purge phase (`internal/app/destroyer_service.go`, `internal/tui/wizard.go`)
- [x] Add explicit OS Config (`osconfig`) explanation and selection in the installer wizard (`internal/tui/wizard.go`)

### Unit 5: Shell Completion
- [x] Configure Zsh and `fzf-tab` file completion to be fully bidirectional case-insensitive (`files/zsh/.base.zsh`)
