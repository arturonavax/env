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
  ✔ ALL TEST SUITES PASSED (4/4 passed in 36s)
══════════════════════════════════════════════════════════════════════
```
