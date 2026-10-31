#!/usr/bin/env bash
# ==============================================================================
# tests/e2e/scripts/test_github_release.sh
# E2E Test: Pre-compiled GitHub release download & extraction (Zero-Go environment)
# ==============================================================================

set -euo pipefail

MOCK_SERVER_URL="${MOCK_SERVER_URL:-http://mock-server:8080}"
DEST_BIN="${HOME}/.local/bin"
REPO_ROOT="${REPO_ROOT:-/repo}"

echo "=== [E2E] Testing GitHub Release Binary Downloader ==="

# Clean environment
rm -rf "${DEST_BIN}/anx" "${DEST_BIN}/aitools" "${DEST_BIN}/safe-suspend" "${HOME}/.config/anx" "${HOME}/.local/state/anx" "${HOME}/.gvm"

echo "-> 1. Running bootstrap in zero-Go environment with ANX_RELEASE_URL..."
export ANX_RELEASE_URL="${MOCK_SERVER_URL}/releases/latest/download/anx_linux_amd64.tar.gz"
export ANX_SUMS_URL="${MOCK_SERVER_URL}/releases/latest/download/SHA256SUMS"

# Deliberately remove /usr/local/go/bin from PATH to guarantee zero-Go environment
export PATH="${DEST_BIN}:/usr/local/bin:/usr/bin:/bin:/sbin:/usr/sbin"
if command -v go >/dev/null 2>&1; then
	echo "[WARN] go command is still available via system path, testing release priority"
fi

# Run bootstrap from the repository
"${REPO_ROOT}/scripts/bootstrap_anx.sh" --yes --profile minimal

echo "-> 2. Verifying release deployment..."
if [[ ! -x "${DEST_BIN}/anx" ]]; then
	echo "[FAIL] ${DEST_BIN}/anx was not deployed from release tarball" >&2
	exit 1
fi

if [[ ! -L "${DEST_BIN}/aitools" ]]; then
	echo "[FAIL] ${DEST_BIN}/aitools symlink was not created" >&2
	exit 1
fi

echo "-> 3. Verifying binary runs without Go installed..."
"${DEST_BIN}/anx" --version
"${DEST_BIN}/anx" doctor

echo "✓ GitHub release downloader test PASSED!"
