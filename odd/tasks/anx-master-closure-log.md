# ANX Master Closure Execution Log

## Overview & Execution Metadata
- **Date:** 2026-10-31
- **Branch:** `audit/apple-silicon-e2e-odd-installer-20261031`
- **Spec:** `odd/new_task_completion.md`
- **Test Gate Status:** 100% Passed (4/4 test suites, 36 Go packages, ShellCheck, Syntax Validation, Integration Smoke).

---

## 1. Phase S01 — Windows Trust & Multi-OS Concurrency

### Problem & Analysis
In Windows CI run `41232029494`, `TestDoctorService_Diagnose` failed because `os.FileMode().Perm() & 0002` assumed POSIX permission bits on Windows. Win32 synthesizes `0777`/`0666` bits from read-only attributes and does not reflect NTFS DACL permissions. Furthermore, `Diagnose` had a side effect of creating directories (`os.MkdirAll`), violating read-only diagnostic guarantees.

### Implemented Solutions
1. **Pluggable Directory Trust Architecture (`internal/security`)**:
   - `DirectoryTrustVerifier` interface and `TrustLevel` (`TrustUnknown`, `TrustTrusted`, `TrustDegraded`, `TrustUntrusted`).
   - `trust_unix.go`: POSIX UID/GID owner inspection with sticky-bit safe ancestry handling.
   - `trust_windows.go`: True Win32 NTFS DACL inspection using `windows.GetNamedSecurityInfo` evaluating `Everyone`, `Anonymous`, `Users`, and `Authenticated Users` with precise write bit masks and inheritance handling.
2. **Read-Only Doctor Service (`internal/app/doctor_service.go`)**:
   - Removed `os.MkdirAll` side effects from `Diagnose`.
   - Returns `VerdictWarn` if the state directory does not exist yet.
   - Remediation never suggests `chmod` on Windows; outputs `icacls` commands.
3. **Evidence**:
   - Unit tests: `TestDoctorService_DiagnoseIsReadOnly`, `TestDoctorService_StateDirectoryVerdictFollowsTrust`, `TestDoctorService_RemediationNeverSuggestsChmodOnWindows`.
   - `go test -race ./internal/security/...` -> 40 passed.
   - `go test -race ./internal/app/...` -> 20 passed.

---

## 2. Phase S02 & S03 — Transactional SQLite Kernel & Zero-Data-Loss Migration

### Problem & Analysis
Legacy `state.json` persistence suffered from multi-process file contention (`os.Rename` failing with `Access is denied` on Windows) and lost records (27/40 records in CI).

### Implemented Solutions
1. **Engine Selection & Architecture (`internal/storage/sqlite`)**:
   - Evaluated `modernc.org/sqlite@v1.61.1` (pure Go, CGO=0) vs alternatives.
   - Documented decision in `docs/adr/ADR-storage-sqlite.md`.
   - Benchmark: 1.04ms cold open, 48.3ms for 10,000 writes in transaction, 12.3MB VmRSS.
   - Configuration: WAL mode (`journal_mode=WAL`), `synchronous=FULL`, `busy_timeout=5000ms`, `foreign_keys=ON`, 2MiB cache.
   - Concurrency: Serialized single-writer pool (`SetMaxOpenConns(1)`), dedicated concurrent reader pool (`SetMaxOpenConns(2)`), and bounded exponential backoff on `SQLITE_BUSY`.
2. **Schema & Tables (`0001_initial.sql`)**:
   - Tables: `metadata`, `resources`, `config_bases`, `applied_hacks`, `host_facts`, `plans`, `plan_operations`, `jobs`, `job_steps`, `events`, `transactions`, `snapshots`, `snapshot_items`, `cas_blobs`, `migration_marker`.
3. **Zero-Data-Loss Migration (`sqlite.ApplyMigration`)**:
   - Preflight check with JSON validation.
   - Timestamped backup: `state.json.pre-sqlite.<timestamp>.bak` (never deleted automatically).
   - Single atomic transaction importing resources, config bases, applied hacks, and metadata.
   - Idempotent re-runs verified.
4. **Adapter Compatibility (`internal/state/store.go`)**:
   - Replaced internal JSON handling in `*state.Store` with SQLite engine while preserving exact method signatures and contracts for existing callers.
5. **CLI State Management (`anx state`)**:
   - `anx state status`: Database, WAL, and entity counts.
   - `anx state migrate [--plan|--apply]`: Migration preview and execution.
   - `anx state backup [--out <path>]`: Online `VACUUM INTO` backup.
   - `anx state verify`: `quick_check` and `foreign_key_check`.
   - `anx state export`: Formatted JSON export.
6. **Evidence**:
   - `TestDB_ConcurrentMultiProcessSimulation` (2, 4, 8 instances concurrently writing): 100% records preserved, 0 lost records.
   - `go test -race ./internal/storage/sqlite/...` -> 12 passed.
   - `go test -race ./internal/state/...` -> 7 passed.

---

## 3. Phase S04 & S05 — Control Plane: Exact Plans, Durable Jobs & Monotonic Events

### Implemented Solutions
1. Integrated `ControlPlaneService.BuildPlan` with immutable `PlanRecord` persistence in SQLite.
2. Implemented durable job lifecycle in `ControlPlaneService.Apply`:
   - State machine: `queued` -> `running` -> `succeeded` / `failed` / `cancelled`.
   - Monotonic audit events emitted to `events` table: `job.started`, `job.succeeded`, `job.failed`, `job.cancelled`.
   - Explicit cancellation support via `ControlPlaneService.Cancel(ctx, jobID)`.
3. New CLI Subcommands:
   - `anx jobs list [--limit <n>]`
   - `anx jobs show <job-id>`
   - `anx jobs events <job-id>`
   - `anx jobs cancel <job-id>`
4. **Evidence**:
   - `TestRepositories_PlansJobsEvents` in SQLite.
   - `TestCLIJobsCommands` in CLI.

---

## 4. Phase S06 — Typed Operation IR & Deterministic Shell Renderers

### Implemented Solutions
1. **Domain Package (`internal/operation`)**:
   - Clean domain package independent of CLI, SQL, or terminal dependencies.
   - Strongly typed `OperationKind`: `OpExecArgv`, `OpEnsureDirectory`, `OpCopyAtomic`, `OpWriteAtomic`, `OpDeleteOwned`, `OpCreateSymlinkOwned`, `OpPackageInstall`, `OpPackageRemove`, `OpPackageUpgrade`, `OpAnxRuntimeDelegation`, etc.
   - Preconditions and Postconditions (`file_exists`, `file_missing`, `hash_equals`).
2. **Defensive Shell Escaping (`internal/operation/escape.go`)**:
   - `EscapePOSIX`: Strictly RFC-compliant single-quote escaping for POSIX `sh` and `bash`. Never uses Go's `%q`.
   - `EscapePowerShell`: Single-quote escaping with `''` for PowerShell.
3. **Deterministic Renderers (`internal/operation/renderer.go`)**:
   - `ShRenderer`: Pure POSIX `sh` with `set -eu`.
   - `BashRenderer`: Bash with `set -euo pipefail`.
   - `PowerShellRenderer`: Windows PowerShell with `$ErrorActionPreference = 'Stop'`.
   - Typed `UnsupportedOperationError` for non-renderable operations.
4. **Native GoExecutor (`internal/operation/executor.go`)**:
   - Executes typed operations natively without shell processes.
   - Verifies preconditions before execution and postconditions after execution.
5. **Consolidated Script Generation**:
   - Consolidated `renderPlanScript` in `internal/cli/packages.go` to delegate directly to `scriptgen.TranslatePlan`.
6. **Evidence**:
   - `TestEscapePOSIX`, `TestEscapePowerShell`, `TestRenderers`, `TestGoExecutor`, `TestCompileIR` in `internal/operation`.

---

## 5. Phase S07 & S08 — Distribution, Cloudflare Worker & Cross-Platform Installers

### Implemented Solutions
1. **Cloudflare Worker Edge Router (`deploy/cloudflare`)**:
   - Routes `sh.arturonavax.dev` (bootstrap) and `env.arturonavax.dev` (docs/landing).
   - User-Agent content negotiation:
     - PowerShell (`pwsh`, `powershell`) -> `install.ps1`.
     - POSIX CLI (`curl`, `wget`, `sh`) -> `install.sh`.
     - Browsers -> HTTP 302 Redirect to `https://arturonavax.dev/env`.
     - Social bots -> Rich OpenGraph preview HTML.
   - Edge caching with `s-maxage=3600` and `stale-while-revalidate=86400`.
   - Unit test suite with `node --test` (4/4 tests passing).
2. **POSIX Installer (`install.sh`)**:
   - Pure POSIX-compliant preflight.
   - Detects architecture, checks releases via SHA256SUMS, standard `go install` fallback, or source clone.
   - Supports `--mode release|source|plan` and `ANX_INSTALL_MODE`.
   - Interactive TTY prompt only when invoked with zero arguments in an interactive terminal.
   - Verified with `sh -n` and `shellcheck`.
3. **Windows Installer (`install.ps1`)**:
   - Source-first bootstrap without requiring Authenticode signing or precompiled EXE assets.
   - Automatically bootstraps repository and compiles via verified official Go archive.

---

## 6. Comprehensive Verification Summary

```text
══════════════════════════════════════════════════════════════════════
  🧪 arturonavax/env Comprehensive Test Harness
══════════════════════════════════════════════════════════════════════

[1/4] Syntax Validation (bash -n)
    ✔ All 33 active shell scripts verified with zero syntax errors.

[2/4] Static Analysis (shellcheck)
    ✔ ShellCheck passed with zero warnings across 18 active scripts.

[3/4] Go Unit & Race Test Suite (go test -race ./...)
    ✔ Go test suite passed with zero data races (36 packages OK).

[4/4] Integration Smoke Suite
    ✔ test_cli_smoke.sh                (PASSED)

══════════════════════════════════════════════════════════════════════
  ✔ ALL TEST SUITES PASSED (4/4 passed in 38s)
══════════════════════════════════════════════════════════════════════
```

---

## 7. Phase F00–F13 — Final Closure & Certification (odd/new_task_completion2.md)

### 7.1 W0-A — Windows DB Lifetime & Resource Cleanup (P0)
- **Problem**: In Windows CI run `41240251525`, `TempDir RemoveAll cleanup: unlinkat ...\anx.db` failed with `Access is denied` because open SQLite handles were held across `app_test.go`, `destroyer_test.go`, `hacks_test.go`, `state_test.go`, and CLI command executions.
- **Fix**:
  1. Made `sqlite.DB.Close()` and `state.Store.Close()` safe and idempotent (`db.writer = nil`, `db.reader = nil`, `s.db = nil`).
  2. Implemented `(c *Container) Close() error` in `internal/bootstrap/bootstrap.go`, ensuring partial initialization failures close open stores cleanly.
  3. Implemented `(a *AppContext) Close() error` and deferred `appCtx.Close()` in `internal/cli/root.go:ExecuteCLI`.
  4. Added `t.Cleanup(func() { _ = store.Close() })` to all fixtures in `app_test.go`, `destroyer_test.go`, `hacks_test.go`, `state_test.go`, and `bootstrap_test.go`.

### 7.2 W0-B — Windows NTFS ACL & Trust Verifier (P0)
- **Problem**: Volume root `D:\` with owner `NETWORK SERVICE` (`S-1-5-20`) was classified as `classOtherPrincipal`, marking ancestors as `TrustUntrusted` and triggering exit 6 in CLI dry-run tests.
- **Fix**:
  1. Classified `WinNetworkServiceSid` (`S-1-5-20`) and `WinLocalServiceSid` (`S-1-5-19`) as `classPrivileged` in `internal/security/trust_windows.go`.
  2. Refined `maskRenameChildren` to `maskDeleteChild | maskWriteDAC | maskWriteOwner | maskGenericAll`, eliminating false positives caused by `maskGenericWrite` on ancestor roots.
  3. Added `WithTrustVerifier` option to `bootstrap.Config` for explicit dependency injection in test fixtures.

### 7.3 CP-ATOMIC — Control Plane Durable Jobs & Atomic Guarantees (P0)
- **Problem**: `control_plane.go` discarded database errors with `_ = db.*`, leading to potential discrepancies between persistent state and executed host actions.
- **Fix**:
  1. Propagated errors in `BuildPlan` and `Apply` for `SavePlan`, `CreateJob`, `AppendEvent`, and `UpdateJobState`.
  2. Enforced pre-mutation atomic transactions: if any DB write fails before external effects, the operation aborts immediately with zero side effects.
  3. Added `needs_reconcile` error state transition if database persistence fails after external execution steps succeed.

### 7.4 IR-UNIFIED — Canonical Operation IR & Topological Execution (P0)
- **Problem**: `OpID` used `step-%d-%s` while `DependsOn` used `step-%d`, resulting in broken DAG edge references. `GoExecutor.ExecutePlan` iterated operations linearly without verifying DAG order.
- **Fix**:
  1. Built two-pass `stepID -> opID` mapping in `internal/operation/compiler.go`, resolving multi-level dependencies to canonical operation IDs.
  2. Implemented `sortOperationsByDAG` using Kahn's algorithm in `internal/operation/executor.go`, guaranteeing topological execution order and detecting cycles or missing dependencies.
  3. Hardened `OpDeleteOwned` and `OpCreateSymlinkOwned` against root path deletion and added error checks for `Sync()` and `Close()` in `writeAtomic`.

### 7.5 INSTALL-P0 & CI Hardening
- **Installer (`install.sh`)**:
  1. Enabled interactive prompt via `/dev/tty` when piped (`curl -fsSL ... | sh`).
  2. Enforced fail-closed 64-char hex checksum validation, aborting if verification tools or hashes are missing.
  3. Enforced safe tar extraction blocking path traversal (`../`) and leading `/`.
  4. Provided structured output in preview mode (`--mode=plan`).
- **CI & E2E**:
  1. Added Cloudflare Worker unit tests (`node --test deploy/cloudflare/src/index.test.js`) to GitHub Actions `lint` job.
  2. Enhanced `tests/e2e/scripts/test_concurrency.sh` with concurrent background multi-process plans and state status checks.
