#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_smart_append.sh
# Unit tests for non-destructive file preservation (smart-append-first).
# Tests marker blocks, idempotent in-place updates, and user customization preservation.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/smart_append.sh"

echo "  Running Smart-Append-First File Preservation Tests..."

setup_sandbox
trap teardown_sandbox EXIT

test_rc="${TEST_SANDBOX}/.zshrc"

# Test 1: New File Creation with Marker Block
smart_append_block "$test_rc" "test-block" "export TEST_VAR=1"
assert_file_exists "$test_rc" "Created new file" &&
	assert_file_contains "$test_rc" "# >>> arturonavax/env:test-block >>>" "Contains begin marker" &&
	assert_file_contains "$test_rc" "export TEST_VAR=1" "Contains payload" &&
	assert_file_contains "$test_rc" "# <<< arturonavax/env:test-block <<<" "Contains end marker" &&
	pass "smart_append_block created new file with marker block"

# Test 2: Appending to Existing File with User Content (User content must be untouched)
user_rc="${TEST_SANDBOX}/user_zshrc"
cat <<'EOF' >"$user_rc"
# User-defined custom aliases
alias mycustom="echo 'custom-user-alias'"
export MY_PERSONAL_TOKEN="secret123"
EOF

smart_append_block "$user_rc" "env-loader" "[[ -f ~/.tools.sh ]] && source ~/.tools.sh"

assert_file_contains "$user_rc" "alias mycustom=" "User alias preserved" &&
	assert_file_contains "$user_rc" "export MY_PERSONAL_TOKEN=" "User secret preserved" &&
	assert_file_contains "$user_rc" "# >>> arturonavax/env:env-loader >>>" "Appended loader marker" &&
	assert_file_contains "$user_rc" "source ~/.tools.sh" "Appended loader body" &&
	pass "smart_append_block preserved existing user configuration and appended block"

# Test 3: Idempotent In-Place Replacement (No duplicate lines or marker proliferation)
smart_append_block "$user_rc" "env-loader" "[[ -f ~/.tools.sh ]] && source ~/.tools.sh # updated payload"

occurrences_begin=$(grep -c "# >>> arturonavax/env:env-loader >>>" "$user_rc")
occurrences_end=$(grep -c "# <<< arturonavax/env:env-loader <<<" "$user_rc")

assert_equals "1" "$occurrences_begin" "Exactly one begin marker exists (no duplicate blocks)" &&
	assert_equals "1" "$occurrences_end" "Exactly one end marker exists (no duplicate blocks)" &&
	assert_file_contains "$user_rc" "updated payload" "Payload updated in-place" &&
	assert_file_contains "$user_rc" "alias mycustom=" "User alias still preserved after update" &&
	pass "smart_append_block updated marker block in-place without duplication"

# Test 4: smart_preserve_copy Preserves Non-Empty Existing File
target_config="${TEST_SANDBOX}/rcmd.conf"
echo "custom_user_rcmd_config=true" >"$target_config"
source_template="${TEST_SANDBOX}/repo_rcmd.conf"
echo "default_repo_config=false" >"$source_template"

smart_preserve_copy "$source_template" "$target_config"

assert_file_contains "$target_config" "custom_user_rcmd_config=true" "Existing user config was not overwritten" &&
	pass "smart_preserve_copy preserved non-empty user file"

# Test 5: smart_preserve_copy Copies When Target is Missing
missing_target="${TEST_SANDBOX}/new_config.conf"
smart_preserve_copy "$source_template" "$missing_target"

assert_file_exists "$missing_target" "Copied to missing target" &&
	assert_file_contains "$missing_target" "default_repo_config=false" "Contains source content" &&
	pass "smart_preserve_copy installed file when target was missing"
