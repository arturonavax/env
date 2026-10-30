# ODD Feature Specification — ANX Control Plane v2 (Phased Evolution)
Branch: `main` | Delivery: `odd-vertical-slices`
Runners: `go test -race ./...` | `go vet ./...` | `bash tests/run_tests.sh --lint`

## Specs

- **S0 — P0 Baseline & Hardening**:
  - Fix `internal/cli/update.go` config drift counting bug (`clean` vs `synced`).
  - Fix error discarding in `internal/app/install_service.go` (`journalAppend`, `recordInstalled`, fail-closed before critical mutations) and `internal/app/sync_service.go` (I/O errors, post-sync command reporting).
  - Consolidate single source of truth for runtime/toolchain documentation (`go.mod`, `scripts/bootstrap_anx.sh`, `README.md`, `AGENTS.md`, `odd/specs/anx/overview.md`).
  - Clean up stale references to `cmd/aitools` in docs.
  - Add GitHub Actions CI workflow (`.github/workflows/ci.yml`) for Linux, macOS, and Windows matrix.
  - Verify baseline reproducibility and sync dry-run simulation without touching `$HOME`.

- **S1 — P1 Core Contracts & Composition Root**:
  - Separate mutable `ExecutionPlan` from immutable `PlanSpec` with deterministic canonical SHA-256 digest (`planDigest = SHA256(canonical(...))`).
  - Define core resource and lifecycle contracts: `ResourceSpec`, `ResourceKind`, `ObservedState`, `Job`, `Event`, `Provenance` in `internal/domain`.
  - Extract reusable composition root from `internal/cli/bootstrap.go` into `internal/bootstrap` with lazy initialization capabilities.
  - Define application ports: `QueryService`, `CommandService`, `EventService` contracts.
  - Implement `anx explain` query provenance foundation.
  - Wire CLI and TUI adapters to shared application interfaces.

- **S2 — P2 Reliable Engine (DAG, Locks & Policies)**:
  - Universal planner with action DAG, fine-grained locks (`pkg:<manager>`, `file:<path>`, `service:<id>`).
  - Dependency-aware scheduler with topological inverse rollback and partial results.
  - Durable journaling and crash recovery verification.

- **S3 — P3 Semantic Configs & Overlays**:
  - Unified sync plan with conflict classification.
  - Semantic adapters preserving comments and formatting.
  - Typed overlays and hardware-aware setting bundles.

- **S4 — P4 Ecosystem, Capabilities & Extensions**:
  - Capability abstraction (`anx ls`, `anx what ls`, `anx explain capability`).
  - Subprocess JSON-RPC v1 protocol for external extensions.
  - Declarative services and AI customizer.

- **S5 — P5 Persistence & Lifecycle**:
  - SQLite transactional store per host with CAS blob storage.
  - Snapshot quotas, recovery state classification.
  - Secure signed self-update and reproducible release pipeline.

- **S6 — P6 Platform & TUI Responsive Refinement**:
  - Responsive multi-breakpoint layouts adhering to the 4 Bubbletea Golden Rules.
  - Multi-OS and Termux support verification.

## Tasks

- [x] **T0.1** (`S0`) | Route: `inline` | Fix `internal/cli/update.go` drift state evaluation bug and add unit/regression tests.
- [x] **T0.2** (`S0`) | Route: `inline` | Propagate journal and store persistence errors in `internal/app/install_service.go` and report post-sync execution in `internal/app/sync_service.go`.
- [x] **T0.3** (`S0`) | Route: `inline` | Align documentation across `README.md`, `AGENTS.md`, `odd/specs/anx/overview.md` (Go version matrix, `cmd/aitools` cleanup).
- [x] **T0.4** (`S0`) | Route: `inline` | Create `.github/workflows/ci.yml` matrix (Linux, macOS, Windows) with lint and race checks.
- [x] **T1.1** (`S1`) | Route: `inline` | Implement immutable `PlanSpec` with canonical SHA-256 `Digest()`, `ExecutionStatus`, `ResourceSpec`, and `ObservedState` in `internal/domain`.
- [x] **T1.2** (`S1`) | Route: `inline` | Extract `internal/bootstrap` composition root from `internal/cli/bootstrap.go`.
- [x] **T1.3** (`S1`) | Route: `inline` | Implement `QueryService` and `CommandService` application interfaces, plus `anx explain` query command foundation.
- [x] **T1.4** (`S1`) | Route: `inline` | Verify P0 + P1 gates (`go test -race ./...`, AC-01..AC-04, AC-11, AC-12, AC-19).

## Log

- **L1 (2026-10-30 — Initiation & Spec Relocation)**:
  Audited `odd/new_tasks.md` vs codebase. Confirmed destination `odd/specs/anx/control-plane-v2.md`. Initialized `odd/tasks/anx-control-plane-v2.md` tracking ODD slices with strict focus on P0 + P1 first.
- **L2 (2026-10-30 — P0 Baseline Hardening Complete)**:
  Fixed config drift classification in `internal/cli/update.go` (`d.State != "clean" && d.State != "synced"`) and added regression tests. Propagated journal and store persistence errors in `internal/app/install_service.go` (fail-closed on begin/commit) and reported post-sync errors in `internal/app/sync_service.go`. Unified documentation for Go versioning and cleaned up stale `cmd/aitools` references while wiring multicall dispatch in `cmd/anx/main.go`. Added GitHub Actions CI matrix (`.github/workflows/ci.yml`). Passed all 166 tests in 30 packages with `-race`.
- **L3 (2026-10-30 — P1 Core Contracts & Composition Root Complete)**:
  Defined immutable `PlanSpec` with deterministic canonical SHA-256 `Digest()`, `ResourceSpec`, `ObservedState`, `DesiredState`, `Job`, `Event`, and `Provenance` in `internal/domain/contracts.go`. Separated execution state from plan specification and proved deterministic digestion in unit tests (AC-03). Extracted Hexagonal composition root into `internal/bootstrap/` decoupled from Cobra/UI dependencies. Implemented `QueryService`, `CommandService`, and `ControlPlaneService` in `internal/app/control_plane.go`. Added `anx explain <resource-id>` command with text and JSON formats (AC-08/AC-09/AC-12). Re-generated troff Section 1 man pages and Markdown CLI documentation. 171 passed tests in 31 packages under `-race`.
