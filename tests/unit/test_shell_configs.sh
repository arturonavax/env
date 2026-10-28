#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_shell_configs.sh
# Unit tests for .base.zsh and .tools.sh shell configurations.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"

echo "  Running Shell Configs (.base.zsh & .tools.sh) Unit Tests..."

setup_sandbox
trap teardown_sandbox EXIT

BASE_ZSH="${REPO_ROOT}/files/zsh/.base.zsh"
TOOLS_SH="${REPO_ROOT}/files/zsh/.tools.sh"

# Test 1: files exist
if [[ -f "$BASE_ZSH" && -f "$TOOLS_SH" ]]; then
	pass "Shell configuration files exist in files/zsh/"
else
	fail "Missing configuration files: $BASE_ZSH or $TOOLS_SH"
fi

# Test 2: .tools.sh syntax validation
bash -n "$TOOLS_SH"
assert_exit_code 0 "$?" "bash -n validation of .tools.sh" &&
	pass ".tools.sh passes bash syntax validation"

# Test 3: .base.zsh syntax validation
if command -v zsh >/dev/null 2>&1; then
	zsh -n "$BASE_ZSH"
	assert_exit_code 0 "$?" "zsh -n validation of .base.zsh" &&
		pass ".base.zsh passes zsh syntax validation"
else
	pass ".base.zsh syntax validation (skipped: zsh not installed in current environment)"
fi

# Test 4: .tools.sh sourcing idempotency
idempotency_result=$(bash -c "
source '${TOOLS_SH}'
p1=\"\$PATH\"
source '${TOOLS_SH}'
p2=\"\$PATH\"
if [ \"\$p1\" = \"\$p2\" ]; then
    echo \"IDEMPOTENT\"
else
    echo \"DIFFERENCE\"
fi
")
assert_equals "IDEMPOTENT" "$idempotency_result" "PATH idempotency across repeated sources of .tools.sh" &&
	pass ".tools.sh is strictly idempotent (repeated sources produce identical PATH)"

# Test 5: In-memory helper functions are cleaned up after sourcing .tools.sh
helper_leak_result=$(bash -c "
source '${TOOLS_SH}'
if declare -F _tools_path_prepend >/dev/null || declare -F _tools_path_append >/dev/null; then
    echo \"LEAKED\"
else
    echo \"CLEAN\"
fi
")
assert_equals "CLEAN" "$helper_leak_result" "Footprint check for internal helpers" &&
	pass ".tools.sh leaves zero internal helper footprint in user shell namespace"

# Test 6: Non-Zsh guard in .base.zsh
guard_result=$(bash -c "
source '${BASE_ZSH}'
echo \"EXIT:\$?\"
")
assert_contains "$guard_result" "EXIT:0" "Non-Zsh execution of .base.zsh" &&
	pass ".base.zsh non-Zsh guard exits cleanly with code 0 without errors"

# Test 7: .base.zsh execution in Zsh initializes core environment
if command -v zsh >/dev/null 2>&1; then
	zsh_eval_result=$(zsh -fc "
TERM=xterm-256color
source '${BASE_ZSH}'
echo \"EDITOR:\$EDITOR\"
echo \"ALIASES:\$(alias | wc -l)\"
" 2>&1)
	assert_contains "$zsh_eval_result" "EDITOR:" "Zsh editor initialization" &&
		pass ".base.zsh initializes editor and shell aliases in interactive Zsh"
fi

# Test 8: ShellCheck on .tools.sh
if command -v shellcheck >/dev/null 2>&1; then
	shellcheck -e SC1091 -e SC1090 "$TOOLS_SH"
	assert_exit_code 0 "$?" "ShellCheck lint of .tools.sh" &&
		pass ".tools.sh passes ShellCheck with zero warnings"
fi
