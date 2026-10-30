#!/usr/bin/env bash
# ==============================================================================
# tests/e2e/scripts/test_cli_and_tui.sh
# E2E Test: Both CLI non-interactive mode and TUI interactive wizard (PTY)
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ANX_BIN="${HOME}/.local/bin/anx"

echo "=== [E2E] Testing CLI (Autonomous) and TUI (Interactive Wizard) ==="

if [[ ! -x "${ANX_BIN}" ]]; then
	echo "[FAIL] anx binary not found at ${ANX_BIN}" >&2
	exit 1
fi

echo "-> 1. Testing CLI non-interactive apply (minimal profile)..."
"${ANX_BIN}" apply --yes --profile minimal

echo "-> 2. Testing CLI dry-run plan output formats (text, json, yaml)..."
"${ANX_BIN}" plan --format text
"${ANX_BIN}" plan --format json
"${ANX_BIN}" plan --format yaml

echo "-> 3. Testing CLI status and diff queries..."
"${ANX_BIN}" status
"${ANX_BIN}" diff

echo "-> 4. Testing TUI interactive wizard via simulated PTY..."
if command -v python3 >/dev/null 2>&1; then
	python3 "${SCRIPT_DIR}/run_wizard_pty.py"
else
	echo "[SKIP] python3 not available for PTY simulation in this image, skipping interactive test"
fi

echo "✓ CLI and TUI testing PASSED!"
