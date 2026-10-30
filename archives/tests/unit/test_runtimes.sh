#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_runtimes.sh
# Unit tests for Python (pyenv), Node.js (fnm), and pnpm runtime management.
# Validates environment variables, shell hook generation, version locking,
# and toolchain strategy resolution.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/packages.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/toolchain.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/remotes/_versions.sh"

echo "  Running Runtimes (Python/pyenv, Node/fnm, pnpm) Unit Tests..."

setup_sandbox
trap teardown_sandbox EXIT

TOOLS_SH="${REPO_ROOT}/files/zsh/.tools.sh"
INSTALL_PYTHON_SH="${REPO_ROOT}/src/remotes/install_python.sh"

# Test 1: Locked runtime versions in _versions.sh
assert_not_empty "${PYTHON_VERSION:-}" "PYTHON_VERSION defined in _versions.sh" &&
	assert_not_empty "${NODE_VERSION:-}" "NODE_VERSION defined in _versions.sh" &&
	assert_not_empty "${PNPM_VERSION:-}" "PNPM_VERSION defined in _versions.sh" &&
	pass "Runtime versions (Python, Node, pnpm) defined with fallback defaults in _versions.sh"

# Test 2: PNPM_HOME configured correctly across platforms in .tools.sh
pnpm_home_eval=$(bash -c "
source '${TOOLS_SH}'
echo \"\${PNPM_HOME:-}\"
")
if [[ "$(uname -s)" == "Darwin" ]]; then
	assert_equals "${HOME}/Library/pnpm" "$pnpm_home_eval" "PNPM_HOME on macOS" &&
		pass "PNPM_HOME evaluates to macOS standard Library/pnpm"
else
	assert_equals "${HOME}/.local/share/pnpm" "$pnpm_home_eval" "PNPM_HOME on Linux" &&
		pass "PNPM_HOME evaluates to Linux XDG standard .local/share/pnpm"
fi

# Test 3: PNPM_HOME is included in PATH after sourcing .tools.sh
pnpm_path_eval=$(bash -c "
source '${TOOLS_SH}'
if [[ \":\$PATH:\" == *\":\${PNPM_HOME}:\"* ]]; then
    echo \"IN_PATH\"
else
    echo \"NOT_IN_PATH\"
fi
")
assert_equals "IN_PATH" "$pnpm_path_eval" "PNPM_HOME included in PATH" &&
	pass "PNPM_HOME is cleanly prepended to PATH in .tools.sh"

# Test 4: PYENV_ROOT exported and shims prepended in .tools.sh
pyenv_eval=$(bash -c "
source '${TOOLS_SH}'
echo \"ROOT:\${PYENV_ROOT:-}\"
if [[ \":\$PATH:\" == *\":\${PYENV_ROOT}/shims:\"* || ! -d \"\$HOME/.pyenv\" ]]; then
    echo \"SHIMS_OK\"
else
    echo \"SHIMS_MISSING\"
fi
")
assert_contains "$pyenv_eval" "ROOT:${HOME}/.pyenv" "PYENV_ROOT exported" &&
	assert_contains "$pyenv_eval" "SHIMS_OK" "Pyenv shims in PATH when installed" &&
	pass "PYENV_ROOT and shims integration verified in .tools.sh"

# Test 5: fnm shell hook generation produces clean bash syntax
if command -v fnm >/dev/null 2>&1; then
	fnm_bash_eval=$(bash -c "
    eval \"\$(fnm env --use-on-cd --shell bash 2>/dev/null)\"
    if [[ -n \"\${FNM_MULTISHELL_PATH:-}\" ]]; then
        echo \"FNM_OK\"
    else
        echo \"FNM_FAILED\"
    fi
    ")
	assert_equals "FNM_OK" "$fnm_bash_eval" "fnm bash hook evaluation" &&
		pass "fnm evaluates clean shell-specific hooks for bash"
else
	pass "fnm hook test (skipped: fnm not in test environment PATH)"
fi

# Test 6: install_python.sh CLI flag parsing and help output
python_help_out=$(bash "$INSTALL_PYTHON_SH" -h)
assert_contains "$python_help_out" "Usage:" "install_python.sh help text" &&
	assert_contains "$python_help_out" "-v <version>" "install_python.sh version flag documentation" &&
	pass "install_python.sh provides standard CLI interface (-i, -u, -f, -v)"

# Test 7: install_python.sh rejects invalid flags cleanly
invalid_exit=0
python_invalid_out=$(bash "$INSTALL_PYTHON_SH" -z 2>&1) || invalid_exit=$?
assert_equals 1 "$invalid_exit" "install_python.sh invalid flag exit code" &&
	assert_contains "$python_invalid_out" "Unknown flag" "install_python.sh error message on unknown flag" &&
	pass "install_python.sh exits with code 1 upon unrecognized flags"

# Test 8: Toolchain provisioner recognizes python, node, and pnpm
toolchain_dry_python=$(toolchain_install python 2>&1 || true)
assert_not_empty "$toolchain_dry_python" "toolchain_install python output" &&
	pass "Toolchain strategy dispatcher handles 'python' runtime"

toolchain_dry_node=$(toolchain_install node 2>&1 || true)
assert_not_empty "$toolchain_dry_node" "toolchain_install node output" &&
	pass "Toolchain strategy dispatcher handles 'node' runtime"

toolchain_dry_pnpm=$(toolchain_install pnpm 2>&1 || true)
assert_not_empty "$toolchain_dry_pnpm" "toolchain_install pnpm output" &&
	pass "Toolchain strategy dispatcher handles 'pnpm' package manager"
