# Archives Directory (`archives/`)

This directory serves as the holding area for decommissioned legacy Bash scripts, utilities, and tests from earlier iterations of `arturonavax/env` before the development and rollout of the unified **ANX Platform** in Go.

---

## Directory Structure

```text
archives/
├── src/                    # Pre-ANX Bash subsystem installers and modular libraries
│   ├── install_terminal.sh # Legacy terminal and shell provisioning script
│   ├── install_editor.sh   # Legacy Neovim and LazyVim setup script
│   ├── install_ai.sh       # Legacy AI tools installer
│   ├── wizard.sh           # Legacy Bash-driven interactive setup wizard
│   ├── _install_requirements.sh # Legacy toolchain requirements check
│   ├── lib/                # Legacy Bash modular libraries (backup, strict, logging, concurrency, etc.)
│   ├── osconfig/           # Legacy desktop environment setup scripts (XFCE, macOS)
│   ├── hacks/              # Legacy hardware hacks (superseded by manifests/hacks/)
│   ├── requirements/       # Legacy pre-flight requirement scripts
│   ├── dom-scraping/       # Legacy scraping helpers
│   └── remotes/            # Legacy remote curl shims
├── utils/                  # Pre-ANX Bash operational utilities
│   ├── sync_config.sh      # Superseded by 'anx sync'
│   ├── backup_config.sh    # Superseded by 'anx backup create'
│   ├── restore_backup.sh   # Superseded by 'anx backup restore'
│   ├── copy_config.sh      # Superseded by 'anx capture'
│   ├── diff_snapshots.sh   # Superseded by 'anx sync --dry-run'
│   ├── rebuild_tools.sh    # Superseded by 'anx rebuild'
│   ├── optimize_system.sh  # Superseded by 'anx optimize'
│   ├── aitools_wizard.sh   # Superseded by 'anx tools ai'
│   └── create_wizard.sh    # Legacy Bash wizard generator
└── tests/                  # Pre-ANX Bash unit and bats test specifications
    ├── unit/               # Legacy Bash unit tests for archives/src/ and archives/utils/
    └── bats/               # Legacy Bats test specs
```

---

## Active Architecture Replacements

Every capability provided by these archived Bash scripts is now natively implemented in Go within `cmd/anx/` and `internal/`:

| Archived Script / Utility | Active ANX Replacement | Description |
|:--------------------------|:-----------------------|:------------|
| `src/wizard.sh` | `anx wizard` / `anx dashboard` | Full-screen Bubble Tea v2 interactive TUI. |
| `src/install_*.sh` | `anx apply [--profile <name>]` | Declarative manifest DAG solver with dependency ordering. |
| `utils/sync_config.sh` | `anx sync [--dry-run]` | Myers unified diff, bidirectional sync, and smart-append markers. |
| `utils/backup_config.sh` | `anx backup create` | Hierarchical snapshot manager with 50MB quota. |
| `utils/restore_backup.sh` | `anx backup restore [-latest]` | Interactive or scripted rollback from snapshots. |
| `utils/copy_config.sh` | `anx capture [group]` | Live dotfile ingestion with ownership-guarded commit/push. |
| `utils/rebuild_tools.sh` | `anx rebuild [all\|lang]` | Multi-runtime recompilation (Go, Rust, Node, Python, Neovim). |
| `utils/optimize_system.sh` | `anx optimize [--report\|--apply]` | Lynis security audit with 12 strict security guardrails. |
| `utils/aitools_wizard.sh` | `anx tools ai [commit\|pr\|llm\|quota]` | Standalone and integrated AI productivity suite. |
| `src/hacks/` | `anx hacks [detect\|apply]` / `manifests/hacks/` | Machine-specific optimizations declared in YAML manifests. |
