#!/usr/bin/env bash
# ==============================================================================
# tests/test_helper.sh
# Production Assertion Library, Test Sandbox Environment & Reporting Utilities.
# Part of arturonavax/env test harness.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_TEST_HELPER_SH_LOADED:-}" ]] && return 0
_TEST_HELPER_SH_LOADED=1

# Resolve repository root
_TESTS_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${_TESTS_DIR}/.." && pwd)"
export REPO_ROOT

# Sourcing repo library
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/colors.sh" 2>/dev/null || true

# Test Runner Accounting
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Create an isolated temporary sandbox directory
setup_sandbox() {
	TEST_SANDBOX="$(mktemp -d "${TMPDIR:-/tmp}/env_test_sandbox.XXXXXX")"
	export TEST_SANDBOX
}

# Destroy the temporary sandbox
teardown_sandbox() {
	if [[ -n "${TEST_SANDBOX:-}" ]] && [[ -d "$TEST_SANDBOX" ]]; then
		rm -rf "$TEST_SANDBOX" 2>/dev/null || true
		unset TEST_SANDBOX
	fi
}

pass() {
	local test_name="$1"
	TESTS_PASSED=$((TESTS_PASSED + 1))
	TESTS_TOTAL=$((TESTS_TOTAL + 1))
	printf '    %s%s%s %s\n' "${GREEN:-}" "${SYM_CHECK:-✔}" "${RESET:-}" "$test_name"
}

fail() {
	local test_name="$1"
	local reason="${2:-Assertion failed}"
	TESTS_FAILED=$((TESTS_FAILED + 1))
	TESTS_TOTAL=$((TESTS_TOTAL + 1))
	printf '    %s%s%s %s\n' "${RED:-}" "${SYM_CROSS:-✖}" "${RESET:-}" "$test_name" >&2
	printf '      %s[REASON]: %s%s\n' "${YELLOW:-}" "$reason" "${RESET:-}" >&2
}

assert_equals() {
	local expected="$1"
	local actual="$2"
	local msg="${3:-Values do not match}"

	if [[ "$expected" != "$actual" ]]; then
		fail "$msg" "Expected: '$expected', Got: '$actual'"
		return 1
	fi
	return 0
}

assert_contains() {
	local haystack="$1"
	local needle="$2"
	local msg="${3:-String does not contain substring}"

	if [[ "$haystack" != *"$needle"* ]]; then
		fail "$msg" "Pattern '$needle' not found in '$haystack'"
		return 1
	fi
	return 0
}

assert_file_exists() {
	local file_path="$1"
	local msg="${2:-File does not exist: $file_path}"

	if [[ ! -e "$file_path" ]]; then
		fail "$msg" "Path not found: $file_path"
		return 1
	fi
	return 0
}

assert_file_contains() {
	local file_path="$1"
	local pattern="$2"
	local msg="${3:-Pattern not found in file}"

	if ! grep -qF "$pattern" "$file_path" 2>/dev/null; then
		fail "$msg" "Pattern '$pattern' not found in $file_path"
		return 1
	fi
	return 0
}

assert_exit_code() {
	local expected_code="$1"
	local actual_code="$2"
	local msg="${3:-Unexpected exit code}"

	if ((expected_code != actual_code)); then
		fail "$msg" "Expected exit code $expected_code, Got $actual_code"
		return 1
	fi
	return 0
}
