#!/usr/bin/env bash
# ==============================================================================
# tests/e2e/scripts/test_curl_install.sh
# E2E Test: Piped remote installer (curl -fsSL ... | bash)
# ==============================================================================

set -euo pipefail

MOCK_SERVER_URL="${MOCK_SERVER_URL:-http://mock-server:8080}"
DEST_BIN="${HOME}/.local/bin"

echo "=== [E2E] Testing Curl Remote Installer Pipeline ==="

# Clean environment before test
rm -rf "${DEST_BIN}/anx" "${DEST_BIN}/aitools" "${HOME}/.config/anx" "${HOME}/.local/state/anx"

echo "-> 1. Fetching and piping install.sh through bash..."
# Use ANX_RELEASE_URL to guarantee fast release extraction during curl pipe
export ANX_RELEASE_URL="${MOCK_SERVER_URL}/releases/latest/download/anx_linux_amd64.tar.gz"
export ANX_REPO_DIR="${HOME}/.local/share/anx/repo"
export ANX_GIT_URL="${REPO_ROOT:-/repo}"

curl -fsSL "${MOCK_SERVER_URL}/install.sh" | bash -s -- -y --profile minimal

echo "-> 2. Verifying compiled binary & multicall symlink..."
if [[ ! -x "${DEST_BIN}/anx" ]]; then
	echo "[FAIL] Expected ${DEST_BIN}/anx to exist and be executable" >&2
	exit 1
fi

if [[ ! -L "${DEST_BIN}/aitools" ]]; then
	echo "[FAIL] Expected ${DEST_BIN}/aitools symlink to exist" >&2
	exit 1
fi

echo "-> 3. Verifying binary execution..."
"${DEST_BIN}/anx" --version
"${DEST_BIN}/anx" status --format json

echo "✓ Curl remote installer test PASSED!"
