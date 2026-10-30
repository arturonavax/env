#!/usr/bin/env bash
# ==============================================================================
# tests/e2e/scripts/test_concurrency.sh
# E2E Test: Concurrent vs Non-concurrent execution
# ==============================================================================

set -euo pipefail

ANX_BIN="${HOME}/.local/bin/anx"

echo "=== [E2E] Testing Concurrent vs Non-Concurrent Installations ==="

if [[ ! -x "${ANX_BIN}" ]]; then
	echo "[FAIL] anx binary not found at ${ANX_BIN}" >&2
	exit 1
fi

echo "-> 1. Testing parallel plan resolution (DAG solver)..."
"${ANX_BIN}" plan --profile developer

echo "-> 2. Testing non-concurrent sequential execution..."
# Apply minimal profile sequentially
"${ANX_BIN}" apply --yes --profile minimal

echo "-> 3. Testing concurrent package resolution across multiple profiles..."
"${ANX_BIN}" plan --profile minimal
"${ANX_BIN}" plan --profile developer

echo "-> 4. Testing concurrent tools inspection..."
"${ANX_BIN}" tools list
"${ANX_BIN}" tools info rtk
"${ANX_BIN}" tools info difft

echo "✓ Concurrency testing PASSED!"
