#!/usr/bin/env bash
# ==============================================================================
# tests/e2e/scripts/test_from_sources.sh
# E2E Test: Build and Install from Source (Go compiler bootstrap & compilation)
# ==============================================================================

set -euo pipefail

DEST_BIN="${HOME}/.local/bin"
REPO_ROOT="${REPO_ROOT:-/repo}"

echo "=== [E2E] Testing Build and Installation from Source ==="

# Force source compilation
export ANX_FORCE_SOURCE=1

echo "-> 1. Compiling ANX from source via bootstrap_anx.sh..."
"${REPO_ROOT}/scripts/bootstrap_anx.sh" --yes --profile minimal

echo "-> 2. Verifying compiled binary & multicall..."
if [[ ! -x "${DEST_BIN}/anx" ]]; then
	echo "[FAIL] Expected compiled ${DEST_BIN}/anx to exist" >&2
	exit 1
fi

"${DEST_BIN}/anx" --version
"${DEST_BIN}/anx" version

echo "-> 3. Testing dry-run plan generation..."
"${DEST_BIN}/anx" plan --profile minimal
"${DEST_BIN}/anx" plan --profile developer

echo "-> 4. Applying profile non-interactively..."
"${DEST_BIN}/anx" apply --yes --profile minimal

echo "✓ Build and Install from Source test PASSED!"
