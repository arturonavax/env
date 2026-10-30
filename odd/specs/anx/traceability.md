# anx — Requirements Traceability Matrix

This matrix maps every high-level requirement to its specification, domain component, Go implementation package, and verification test.

| Req ID | Description | Spec Document | Package / Component | Verification Test |
|---|---|---|---|---|
| **ARCH-001** | Hexagonal Architecture & zero TUI/CLI deps in domain | `odd/specs/anx/architecture.md` | `internal/domain` | `TestDomainHermetic` |
| **ARCH-002** | Layer separation (domain, app, adapters, presentation) | `odd/specs/anx/architecture.md` | `internal/app` | Go package import rules |
| **ARCH-004** | Explicit dependency injection & zero mutable globals | `odd/specs/anx/architecture.md` | `internal/app`, `internal/cli` | Constructor test suites |
| **PLAT-001** | Linux distribution detection & package manager mapping | `odd/specs/anx/platform-compatibility.md` | `internal/platform` | `TestPlatformLinuxDetection` |
| **PLAT-002** | macOS Apple Silicon & Intel detection | `odd/specs/anx/platform-compatibility.md` | `internal/platform` | `TestPlatformDarwinDetection` |
| **PLAT-003** | Native Windows support without Bash/WSL | `odd/specs/anx/platform-compatibility.md` | `internal/platform` | `TestPlatformWindowsDetection` |
| **PLAT-005** | Android Termux environment detection & pkg paths | `odd/specs/anx/platform-compatibility.md` | `internal/platform` | `TestPlatformTermuxDetection` |
| **CLI-001** | Root & subcommands hierarchy via Cobra | `odd/specs/anx/cli-contract.md` | `internal/cli` | `TestCobraCommandTree` |
| **CLI-003** | Machine-readable JSON output on `--format json` | `odd/specs/anx/cli-contract.md` | `internal/cli` | `TestCommandJSONOutput` |
| **TUI-001** | Bubbletea v2 Elm architecture & 4 Golden Rules | `odd/specs/anx/tui-design.md` | `internal/tui` | `TestTUIModelUpdate` |
| **TUI-002** | Tokyo Night semantic color tokens | `odd/specs/anx/tui-design.md` | `internal/theme` | `TestTokyoNightTokens` |
| **TUI-003** | Responsive layouts (wide, medium, narrow) | `odd/specs/anx/tui-design.md` | `internal/tui` | `TestTUIResponsiveResize` |
| **PKG-001** | Declarative package catalog with multi-provider support | `odd/specs/anx/manifest-schema.md` | `internal/catalog` | `TestCatalogValidation` |
| **CFG-001** | True Three-Way dotfile merge (base, local, incoming) | `odd/specs/anx/synchronization.md` | `internal/configsync` | `TestThreeWayMerge` |
| **CFG-002** | Smart-append shell marker block preservation | `odd/specs/anx/synchronization.md` | `internal/configsync` | `TestMarkerPreservation` |
| **TXN-001** | Plan generation prior to execution | `odd/specs/anx/state-and-transactions.md` | `internal/domain`, `internal/app` | `TestExecutionPlanGeneration` |
| **TXN-002** | Append-only JSONL transaction journal | `odd/specs/anx/state-and-transactions.md` | `internal/journal` | `TestJournalAppendAndRead` |
| **TXN-003** | Crash recovery & `anx resume` | `odd/specs/anx/state-and-transactions.md` | `internal/journal` | `TestInterruptedTxRecovery` |
| **LIFE-001** | `anx update` non-mutating index refresh | `odd/specs/anx/overview.md` | `internal/app` | `TestUpdateSemantics` |
| **LIFE-002** | `anx upgrade` scoped tool upgrading | `odd/specs/anx/overview.md` | `internal/app` | `TestUpgradePlanning` |
| **LIFE-003** | `anx sync` dotfile reconciliation | `odd/specs/anx/synchronization.md` | `internal/app` | `TestSyncReconciliation` |
| **SEC-001** | Safe process execution without shell string injection | `odd/specs/anx/security.md` | `internal/security` | `TestSafeProcessRunner` |
| **SEC-003** | OSV vulnerability scanner integration | `odd/specs/anx/security.md` | `internal/security` | `TestOSVAudit` |
| **PERF-001** | Cold start latency budgets | `odd/specs/anx/overview.md` | `cmd/anx` | `BenchmarkColdStart` |
| **QA-001** | High test coverage across core domain | `odd/specs/anx/testing-and-acceptance.md` | `internal/...` | `go test ./...` |
