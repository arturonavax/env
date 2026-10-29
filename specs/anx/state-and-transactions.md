# anx — State, Transactions, Journals & Crash Recovery

## 1. Local State Directory Layout

`anx` stores state under the standard platform state directory:
- Linux: `~/.local/state/anx/`
- macOS: `~/Library/Application Support/anx/`
- Windows: `%LOCALAPPDATA%\anx\`

```text
~/.local/state/anx/
├── journal.jsonl           # Append-only transaction log
├── state.json              # Durable state: installed packages, active profile, sync bases
├── backups/                # Managed backups and snapshots (50MB quota rotated)
│   ├── 2026-10-29_12-00-00/
│   │   ├── manifest.json
│   │   └── files/
│   └── ...
└── lock                    # Cross-platform advisory file lock
```

## 2. Transaction Lifecycle & Journaling

Each transaction follows a strict state machine:
`Planned` -> `Validated` -> `Executing` -> `Committed` (or `RolledBack` / `Interrupted`)

Every operation appends a JSON line to `journal.jsonl`:
```json
{"tx_id":"tx-20261029-01","timestamp":"2026-10-29T12:00:00Z","type":"begin","plan_id":"plan-01","steps":3}
{"tx_id":"tx-20261029-01","timestamp":"2026-10-29T12:00:01Z","type":"step_start","step_id":1,"action":"backup_file","target":"~/.zshrc"}
{"tx_id":"tx-20261029-01","timestamp":"2026-10-29T12:00:02Z","type":"step_done","step_id":1,"backup_path":"~/.local/state/anx/backups/.../zshrc"}
{"tx_id":"tx-20261029-01","timestamp":"2026-10-29T12:00:03Z","type":"step_start","step_id":2,"action":"smart_append","target":"~/.zshrc"}
{"tx_id":"tx-20261029-01","timestamp":"2026-10-29T12:00:04Z","type":"step_done","step_id":2}
{"tx_id":"tx-20261029-01","timestamp":"2026-10-29T12:00:05Z","type":"commit"}
```

## 3. Crash Recovery & Resume

When `anx` starts:
1. It inspects `journal.jsonl`.
2. If the last transaction is not marked `commit` or `rolled_back`, it is flagged as `Interrupted`.
3. The user is prompted (or `anx resume` is invoked) to either:
   - **Resume**: Check remaining unapplied steps and continue forward idempotently.
   - **Rollback**: Execute registered compensating actions in LIFO order using the pre-mutation backups.
