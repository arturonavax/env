# anx — Security Architecture & Vulnerability Auditing

## 1. Process Execution Hardening

To avoid shell injection, race conditions, and uncontrolled execution:
- **No Shell Execution**: Processes are invoked strictly via `exec.CommandContext(ctx, binary, args...)` with discrete argument slices. Never call `sh -c` or `bash -c` with string-concatenated commands.
- **Strict Context Timeouts**: Every external invocation has a configurable timeout (default: 60s for package managers, 15s for version checks).
- **Environment Scrubbing**: Sensitive environment variables (`GITHUB_TOKEN`, `SSH_AUTH_SOCK`, `API_KEY`) are protected and never logged in plain text.
- **Output Truncation**: Process stdout/stderr capture is capped at 1MB to avoid memory exhaustion from runaway commands.

## 2. Filesystem Protection & Path Traversal Prevention

- **Path Canonicalization**: All file paths are cleaned via `filepath.Clean` and evaluated for symlink targets via `filepath.EvalSymlinks`.
- **Root Jail**: Dotfiles cannot be written outside the user's home directory (`$HOME` / `%USERPROFILE%`) unless an explicit system integration strategy is declared and approved.
- **Safe Atomic Replacement**: Files are written to temporary files on the same filesystem (`.anx-tmp-XXXXXX`) and renamed into place atomically via `os.Rename`.

## 3. Vulnerability Auditing (`anx audit`)

`anx audit` provides continuous security monitoring for all managed CLI tools and runtimes:
- **Source**: Queries the Open Source Vulnerabilities (OSV) API (`https://api.osv.dev/v1/query`) using package names, ecosystems, and detected semantic versions.
- **Offline Cache**: Advisory database cached locally under `~/.local/state/anx/advisories/` to allow offline audits.
- **Severity Classification**: Findings normalized to CVSS v3 ratings (Critical, High, Medium, Low) with clear remediating upgrade commands.
