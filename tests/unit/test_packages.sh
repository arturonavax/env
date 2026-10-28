#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_packages.sh
# Unit tests for multi-platform package translation matrix.
# Tests Debian, Fedora, Arch Linux, macOS, openSUSE, and Termux translations.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/packages.sh"

echo "  Running Package Translation Matrix Tests..."

# Test 1: Debian / Ubuntu Translations
assert_equals "build-essential" "$(_translate_debian "build-essential")" "Debian build-essential" &&
	assert_equals "libssl-dev" "$(_translate_debian "ssl-dev")" "Debian ssl-dev" &&
	assert_equals "libsqlite3-dev" "$(_translate_debian "sqlite-dev")" "Debian sqlite-dev" &&
	assert_equals "keyd" "$(_translate_debian "keyd")" "Debian keyd" &&
	assert_equals "ripgrep" "$(_translate_debian "ripgrep")" "Debian ripgrep" &&
	pass "Debian/Ubuntu translation matrix verified"

# Test 2: Fedora / RHEL Translations
assert_equals "@development-tools" "$(_translate_fedora "build-essential")" "Fedora build-essential" &&
	assert_equals "openssl-devel" "$(_translate_fedora "ssl-dev")" "Fedora ssl-dev" &&
	assert_equals "sqlite sqlite-devel" "$(_translate_fedora "sqlite-dev")" "Fedora sqlite-dev" &&
	assert_equals "keyd" "$(_translate_fedora "keyd")" "Fedora keyd" &&
	assert_equals "ripgrep" "$(_translate_fedora "ripgrep")" "Fedora ripgrep" &&
	pass "Fedora/RHEL translation matrix verified"

# Test 3: Arch Linux Translations
assert_equals "base-devel" "$(_translate_arch "build-essential")" "Arch build-essential" &&
	assert_equals "openssl" "$(_translate_arch "ssl-dev")" "Arch ssl-dev" &&
	assert_equals "sqlite" "$(_translate_arch "sqlite-dev")" "Arch sqlite-dev" &&
	assert_equals "keyd" "$(_translate_arch "keyd")" "Arch keyd" &&
	assert_equals "python" "$(_translate_arch "python3")" "Arch python3" &&
	assert_equals "ripgrep" "$(_translate_arch "ripgrep")" "Arch ripgrep" &&
	pass "Arch Linux translation matrix verified"

# Test 4: macOS Darwin (Homebrew) Translations
assert_equals "" "$(_translate_darwin "build-essential")" "macOS build-essential (built into CLT)" &&
	assert_equals "openssl@3" "$(_translate_darwin "ssl-dev")" "macOS ssl-dev" &&
	assert_equals "sqlite" "$(_translate_darwin "sqlite-dev")" "macOS sqlite-dev" &&
	assert_equals "" "$(_translate_darwin "keyd")" "macOS keyd (N/A on Darwin)" &&
	assert_equals "" "$(_translate_darwin "wmctrl")" "macOS wmctrl (N/A on Darwin)" &&
	assert_equals "python@3" "$(_translate_darwin "python3")" "macOS python3" &&
	assert_equals "ripgrep" "$(_translate_darwin "ripgrep")" "macOS ripgrep" &&
	pass "macOS Darwin translation matrix verified (Linux-only skipped cleanly)"

# Test 5: openSUSE (zypper) Translations
assert_equals "devel_basis" "$(_translate_suse "build-essential")" "SUSE build-essential" &&
	assert_equals "libopenssl-devel" "$(_translate_suse "ssl-dev")" "SUSE ssl-dev" &&
	assert_equals "sqlite3-devel" "$(_translate_suse "sqlite-dev")" "SUSE sqlite-dev" &&
	assert_equals "keyd" "$(_translate_suse "keyd")" "SUSE keyd" &&
	assert_equals "ripgrep" "$(_translate_suse "ripgrep")" "SUSE ripgrep" &&
	pass "openSUSE / SLES translation matrix verified"

# Test 6: Android Termux (pkg) Translations
assert_equals "build-essential" "$(_translate_termux "build-essential")" "Termux build-essential" &&
	assert_equals "openssl" "$(_translate_termux "ssl-dev")" "Termux ssl-dev" &&
	assert_equals "sqlite" "$(_translate_termux "sqlite-dev")" "Termux sqlite-dev" &&
	assert_equals "" "$(_translate_termux "keyd")" "Termux keyd (N/A on Android)" &&
	assert_equals "clang" "$(_translate_termux "gcc")" "Termux gcc (uses clang)" &&
	assert_equals "python" "$(_translate_termux "python3")" "Termux python3" &&
	assert_equals "ripgrep" "$(_translate_termux "ripgrep")" "Termux ripgrep" &&
	pass "Android Termux translation matrix verified"

# Test 7: Fallback for Unknown Package Names
assert_equals "custom-binary-tool" "$(pkg_translate "custom-binary-tool")" "Unknown package passthrough" &&
	pass "Unknown package passthrough verified"
