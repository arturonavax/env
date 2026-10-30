# anx — System Overview & Architecture Spec

## 1. Executive Summary

`anx` is a unified, high-performance, cross-platform CLI and TUI written in Go for developer environment and dotfiles management. It evolves the Bash-based orchestration of `arturonavax/env` into an enterprise-grade modular monolith adhering to Spec-Driven Development (SDD), Hexagonal Architecture, Declarative Desired State, and Transactional Execution.

## 2. Fundamental Product Invariants

- **INV-001 — One Unified Application**: `anx` is a single Go binary providing both CLI and TUI over the same shared domain services. Zero duplicate logic.
- **INV-002 — Zero Mandatory Bash Runtime Dependency**: Core operations run natively on Windows (no WSL required), macOS, Linux, and Android/Termux without requiring Bash, Python, or Node.js.
- **INV-003 — Core Portability Independent of Package Portability**: `anx` boots, renders UI, inspects state, and manages configs on every platform even if certain external tools are platform-restricted.
- **INV-004 — Non-Destructive by Default**: No user data is overwritten or deleted without validated plans, backups, ownership tracking, and explicit confirmation.
- **INV-005 — Deterministic Operation**: Equivalent manifests and state produce identical plans across runs.
- **INV-006 — Repeatability and Idempotency**: Applying an already satisfied desired state incurs zero mutations.
- **INV-007 — CLI/TUI Parity**: Every action achievable in the TUI is exposed via the CLI or headless flags.
- **INV-008 — Offline-First and Private**: Local operations require zero network access. No telemetry, external accounts, or mandatory cloud subscriptions.
- **INV-009 — Honest Recovery Semantics**: External package operations that cannot be undone are explicitly flagged with compensation guidance rather than claiming false atomicity.
- **INV-010 — Measured Performance**: Cold start < 15ms (CLI) / < 30ms (TUI), memory < 25MB idle, zero CPU polling when idle.

## 3. Technology Stack

- **Language**: Go 1.26+ (built with modern toolchains, compatible with bootstrap runtime Go >= 1.24)
- **CLI Framework**: `github.com/spf13/cobra` (v1.10.2)
- **TUI Framework**: `charm.land/bubbletea/v2` (v2.1.1)
- **Styling**: `charm.land/lipgloss/v2` (v2.0.6)
- **Components**: `charm.land/bubbles/v2` (v2.2.1)
- **Serialization**: `gopkg.in/yaml.v3`, standard `encoding/json`
- **Theme**: Tokyo Night (default palette `#1a1b26`, `#c0caf5`, `#7aa2f7`, `#7dcfff`, `#bb9af7`, `#9ece6a`, `#e0af68`, `#f7768e`, `#565f89`)
