#!/usr/bin/env bats
# ==============================================================================
# tests/bats/smart_append.bats
# Bats specification for smart non-destructive file preservation.
# ==============================================================================

setup() {
	REPO_ROOT="$(cd -- "$BATS_TEST_DIRNAME/../.." && pwd)"
	# shellcheck source=/dev/null
	source "$REPO_ROOT/src/lib/smart_append.sh"
	TEST_SANDBOX="$(mktemp -d "${TMPDIR:-/tmp}/bats_sandbox.XXXXXX")"
}

teardown() {
	rm -rf "$TEST_SANDBOX"
}

@test "smart_append_block creates marker block in new file" {
	local test_file="$TEST_SANDBOX/test.rc"
	smart_append_block "$test_file" "test-tag" "echo 'hello'"
	[ -f "$test_file" ]
	grep -qF "# >>> arturonavax/env:test-tag >>>" "$test_file"
	grep -qF "echo 'hello'" "$test_file"
	grep -qF "# <<< arturonavax/env:test-tag <<<" "$test_file"
}

@test "smart_append_block preserves custom user lines" {
	local test_file="$TEST_SANDBOX/user.rc"
	echo "alias custom=true" > "$test_file"
	smart_append_block "$test_file" "loader" "echo 'managed'"
	grep -qF "alias custom=true" "$test_file"
	grep -qF "echo 'managed'" "$test_file"
}

@test "smart_append_block updates existing block in-place without duplicating" {
	local test_file="$TEST_SANDBOX/user.rc"
	smart_append_block "$test_file" "loader" "v1"
	smart_append_block "$test_file" "loader" "v2"

	local count
	count="$(grep -c "# >>> arturonavax/env:loader >>>" "$test_file")"
	[ "$count" -eq 1 ]
	grep -qF "v2" "$test_file"
	! grep -qF "v1" "$test_file"
}
