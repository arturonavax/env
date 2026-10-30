#!/usr/bin/env bash
# ==============================================================================
# tests/e2e/scripts/test_config_and_tools.sh
# E2E Test: Dotfile sets, 3-way reconciliation, personal tools, and doctor
# ==============================================================================

set -euo pipefail

ANX_BIN="${HOME}/.local/bin"

echo "=== [E2E] Testing Dotfile Sets, Personal Tools & Health Doctor ==="

if [[ ! -x "${ANX_BIN}/anx" ]]; then
	echo "[FAIL] anx binary not found at ${ANX_BIN}/anx" >&2
	exit 1
fi

echo "-> 1. Testing dotfiles sync dry-run..."
"${ANX_BIN}/anx" sync --dry-run

echo "-> 2. Testing dotfiles live reconciliation..."
"${ANX_BIN}/anx" sync

echo "-> 3. Verifying dotfile diff reports clean/up-to-date state..."
"${ANX_BIN}/anx" diff

echo "-> 4. Testing hierarchical configuration backup engine..."
"${ANX_BIN}/anx" backup create "e2e-test-backup"
"${ANX_BIN}/anx" backup list

echo "-> 5. Testing personal tools engine & AI shortcuts..."
"${ANX_BIN}/anx" tools list
"${ANX_BIN}/anx" tools ai --help
"${ANX_BIN}/anx" ai --help

echo "-> 6. Testing cleanup and orphan detection in dry-run mode..."
"${ANX_BIN}/anx" cleanup --dry-run

echo "-> 7. Running comprehensive system health doctor..."
"${ANX_BIN}/anx" doctor

echo "✓ Configuration sets, personal tools and diagnostics testing PASSED!"
