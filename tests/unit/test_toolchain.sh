#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_toolchain.sh
# Unit tests for multi-tier toolchain fallback architecture and dependency resolution.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/toolchain.sh"

echo "  Running Toolchain & Multi-Tier Fallback Tests..."

setup_sandbox
trap teardown_sandbox EXIT

# Test 1: Strategy Constants Defined
assert_equals "binary" "$STRATEGY_BINARY" "Strategy constant: binary" &&
	assert_equals "package" "$STRATEGY_PACKAGE" "Strategy constant: package" &&
	assert_equals "script" "$STRATEGY_SCRIPT" "Strategy constant: script" &&
	assert_equals "source" "$STRATEGY_SOURCE" "Strategy constant: source" &&
	pass "Multi-tier strategy constants properly exported"

# Test 2: Dependency Resolver for Empty List
empty_deps_output="$(toolchain_ensure_dependencies "empty_tool" 2>&1)"
assert_equals "" "$empty_deps_output" "Empty dependency list returns cleanly" &&
	pass "toolchain_ensure_dependencies handles zero-dependency tools without overhead"

# Test 3: Idempotent Tool Installation (Golang)
# Since Go is already installed on this machine, toolchain_install should detect it instantly
go_install_output="$(toolchain_install golang 2>&1)"
go_install_code=$?
assert_exit_code 0 "$go_install_code" "toolchain_install golang exit code" &&
	assert_contains "$go_install_output" "Golang is already installed" "Existing Go detection message" &&
	pass "toolchain_install detected existing Golang installation idempotently"

# Test 4: Unknown Tool Reporting
unknown_output="$(toolchain_install nonexistent_sample_tool 2>&1 || true)"
assert_contains "$unknown_output" "Unknown toolchain tool" "Reports unknown tool error" &&
	pass "toolchain_install properly handles unknown tool requests"
