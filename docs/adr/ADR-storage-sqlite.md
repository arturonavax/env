# ADR-storage-sqlite: Transactional Embedded SQLite Storage Engine for ANX

- **Status:** Accepted
- **Deciders:** ANX Core Architecture Team
- **Date:** 2026-10-31

## Context

`anx` previously persisted state (installed resources, configuration ancestor bases for 3-way merges, applied hardware hacks, active profiles) in a single JSON file (`state.json`) managed via file locks and atomic file replacement (`os.Rename`).

### Failure Modes of JSON + FileLock
1. **Multi-process concurrency under Windows:** Win32 denies file replacement (`os.Rename` -> `Access is denied`) when another process holds a read handle or when antivirus/indexers inspect the file. In CI run `41232029494`, `TestStoreMultiProcessSimulation` resulted in 27/40 records due to clobbered write attempts.
2. **Crash safety:** If a crash occurs during in-memory state mutations or partial disk flushes, state is vulnerable to truncation or desynchronization.
3. **Memory footprint and scaling:** Loading all historical config bases and resource records into RAM on every CLI invocation consumes unnecessary heap and degrades cold-start latency.
4. **Lack of queryability:** No indexing or structured querying for control plane operations, durable background jobs, execution events, or CAS blob reachability.

## Decision

We migrate ANX state persistence from `state.json` to an embedded SQLite transactional engine using pure Go driver **`modernc.org/sqlite`** via standard Go `database/sql`.

### Evaluated Alternatives

1. **`modernc.org/sqlite` (Selected):**
   - **CGO:** Pure Go (CGO=0), transpiled from SQLite via `ccgo`.
   - **Cross-compilation:** Native support for `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, `windows/{amd64,arm64}` out of the box with zero external toolchains.
   - **Cold Start & Latency:** 1.04ms open+create, 48.3ms for 10,000 writes in transaction.
   - **Memory (RSS):** ~12.3 MiB resident memory.
   - **Driver Interface:** Idiomatic Go `database/sql`.
   - **Limitation:** Android Termux (Bionic) is not officially targeted upstream; marked conditional until tested on physical Bionic environment.

2. **`github.com/ncruces/go-sqlite3`:**
   - Transpiled to WebAssembly with wazero runtime. Good VFS isolation, but higher runtime memory overhead and additional layer of abstraction.

3. **`mattn/go-sqlite3`:**
   - Requires CGO, breaking pure Go cross-compilation across platforms without cross-compilers (musl, Windows MinGW, macOS cross-SDKs).

## Architecture & Configuration

- **File Location:** `filepath.Join(stateDir, "anx.db")`.
- **Permissions:** `0700` directory, `0600` database file on POSIX; restricted DACL on Windows.
- **Journal Mode:** Write-Ahead Logging (`WAL`).
- **Synchronous:** `FULL` by default for durable power-loss protection.
- **Busy Timeout:** `5000ms` with bounded exponential backoff.
- **Connection Pool:** Serialized writer pool (`SetMaxOpenConns(1)`), dedicated read connection pool (`SetMaxOpenConns(2)`), preventing write lock contention.
- **Migration & Zero Data Loss:** Automatic, idempotent migration from existing `state.json` with timestamped backup `state.json.pre-sqlite.<ts>.bak` and integrity verification before marking complete.
- **Control Plane Integration:** Tables for `resources`, `config_bases`, `applied_hacks`, `plans`, `plan_operations`, `jobs`, `job_steps`, `events`, and `cas_blobs`.

## Consequences

- Multi-process concurrent operations across Windows, macOS, and Linux are completely race-free and ACID-compliant.
- `state.json` is safely deprecated and migrated without user intervention or data loss.
- High-performance indexed lookups for orphaned resources, config 3-way merges, and plan authorization.
