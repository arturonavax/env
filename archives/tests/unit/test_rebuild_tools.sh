#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_rebuild_tools.sh
# Unit tests for universal runtime post-upgrade tool rebuilder & synchronizer.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"

echo "  Running Universal Tool Rebuilder (rebuild_tools.sh) Unit Tests..."

setup_sandbox
trap teardown_sandbox EXIT

REBUILD_SCRIPT="${REPO_ROOT}/utils/rebuild_tools.sh"
REMOTE_SCRIPT="${REPO_ROOT}/src/remotes/rebuild_tools.sh"

# Test 1: Files exist and are executable
assert_file_exists "$REBUILD_SCRIPT" "utils/rebuild_tools.sh exists" &&
	[[ -x "$REBUILD_SCRIPT" ]] &&
	assert_file_exists "$REMOTE_SCRIPT" "src/remotes/rebuild_tools.sh exists" &&
	[[ -x "$REMOTE_SCRIPT" ]] &&
	pass "rebuild_tools.sh scripts exist and have execute permissions"

# Test 2: Help flag output
help_out=$(bash "$REBUILD_SCRIPT" --help)
assert_contains "$help_out" "Usage:" "Help output usage header" &&
	assert_contains "$help_out" "--python" "Python flag documented" &&
	assert_contains "$help_out" "--node" "Node flag documented" &&
	assert_contains "$help_out" "--go" "Go flag documented" &&
	assert_contains "$help_out" "--rust" "Rust flag documented" &&
	assert_contains "$help_out" "--nvim" "Neovim flag documented" &&
	assert_contains "$help_out" "--dry-run" "Dry-run flag documented" &&
	pass "rebuild_tools.sh provides comprehensive CLI help text"

# Test 3: Full dry-run execution
dry_run_out=$(bash "$REBUILD_SCRIPT" --dry-run)
assert_contains "$dry_run_out" "Universal Tool Rebuilder" "Dry run header" &&
	assert_contains "$dry_run_out" "[1] Python Runtime" "Python stage in dry-run" &&
	assert_contains "$dry_run_out" "[2] Node.js" "Node stage in dry-run" &&
	assert_contains "$dry_run_out" "[3] Golang Developer Tools" "Go stage in dry-run" &&
	assert_contains "$dry_run_out" "[4] Rust Toolchain" "Rust stage in dry-run" &&
	assert_contains "$dry_run_out" "[5] Neovim Plugins" "Neovim stage in dry-run" &&
	assert_contains "$dry_run_out" "Universal Tool Rebuild Completed" "Dry run completion summary" &&
	pass "rebuild_tools.sh --dry-run previews all 5 subsystems cleanly"

# Test 4: Subsystem targeting (Python only)
py_dry_out=$(bash "$REBUILD_SCRIPT" --python --dry-run)
assert_contains "$py_dry_out" "[1] Python Runtime" "Python target output" &&
	[[ "$py_dry_out" != *"[2] Node.js"* ]] &&
	[[ "$py_dry_out" != *"[3] Golang Developer Tools"* ]] &&
	pass "rebuild_tools.sh --python targets exclusively Python subsystem"

# Test 5: Subsystem targeting (Go only)
go_dry_out=$(bash "$REBUILD_SCRIPT" --go --dry-run)
assert_contains "$go_dry_out" "[3] Golang Developer Tools" "Go target output" &&
	[[ "$go_dry_out" != *"[1] Python Runtime"* ]] &&
	[[ "$go_dry_out" != *"[2] Node.js"* ]] &&
	pass "rebuild_tools.sh --go targets exclusively Golang subsystem"

# Test 6: Subsystem targeting (Node only)
node_dry_out=$(bash "$REBUILD_SCRIPT" --node --dry-run)
assert_contains "$node_dry_out" "[2] Node.js" "Node target output" &&
	[[ "$node_dry_out" != *"[1] Python Runtime"* ]] &&
	[[ "$node_dry_out" != *"[3] Golang Developer Tools"* ]] &&
	pass "rebuild_tools.sh --node targets exclusively Node.js subsystem"

# Test 7: Invalid flag rejection
invalid_code=0
invalid_out=$(bash "$REBUILD_SCRIPT" --bogus-flag 2>&1) || invalid_code=$?
assert_equals 1 "$invalid_code" "Invalid option exit code" &&
	assert_contains "$invalid_out" "Unknown option" "Rejection notice on invalid flag" &&
	pass "rebuild_tools.sh rejects invalid flags with exit code 1"

# Test 8: Remote wrapper execution in dry-run
remote_dry_out=$(bash "$REMOTE_SCRIPT" --dry-run)
assert_contains "$remote_dry_out" "Universal Tool Rebuilder" "Remote wrapper forwarding" &&
	pass "src/remotes/rebuild_tools.sh successfully forwards to utils/rebuild_tools.sh"
