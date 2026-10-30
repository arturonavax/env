#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_platform.sh
# Unit tests for platform discovery, architecture, and Termux/Windows emulation.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/platform.sh"

echo "  Running Platform Discovery Tests..."

# Test 1: Operating System Detection
current_os="$(detect_os)"
if [[ "$current_os" =~ ^(linux|darwin|wsl|android|windows_msys|windows_cygwin)$ ]]; then
	pass "detect_os returned valid platform: '$current_os'"
else
	fail "detect_os returned invalid platform: '$current_os'"
fi

# Test 2: Architecture Detection
current_arch="$(detect_arch)"
if [[ "$current_arch" =~ ^(x86_64|arm64|armv7|x86_32)$ ]]; then
	pass "detect_arch returned valid CPU architecture: '$current_arch'"
else
	fail "detect_arch returned unexpected architecture: '$current_arch'"
fi

# Test 3: Distribution Family Detection
current_distro="$(detect_distro)"
if [[ -n "$current_distro" && "$current_distro" != "unknown" ]]; then
	pass "detect_distro detected distribution: '$current_distro'"
else
	fail "detect_distro failed to identify distribution: '$current_distro'"
fi

# Test 4: Package Manager Detection
current_pkg_mgr="$(detect_pkg_mgr)"
if [[ "$current_pkg_mgr" =~ ^(apt|dnf|yum|pacman|zypper|apk|brew|termux)$ ]]; then
	pass "detect_pkg_mgr detected supported package manager: '$current_pkg_mgr'"
else
	fail "detect_pkg_mgr returned unsupported manager: '$current_pkg_mgr'"
fi

# Test 5: Emulated Android (Termux) Sandbox Detection
(
	export TERMUX_VERSION="0.118.0"
	export PREFIX="/data/data/com.termux/files/usr"
	os="$(detect_os)"
	distro="$(detect_distro)"
	pkg_mgr="$(detect_pkg_mgr)"
	has_sudo=$(has_sudo_capability && echo true || echo false)

	assert_equals "android" "$os" "Termux emulation: detect_os" &&
		assert_equals "termux" "$distro" "Termux emulation: detect_distro" &&
		assert_equals "termux" "$pkg_mgr" "Termux emulation: detect_pkg_mgr" &&
		assert_equals "false" "$has_sudo" "Termux emulation: has_sudo_capability"
) && pass "Android Termux emulation detected correctly without sudo"

# Test 6: Emulated Windows (MSYS / Git Bash) Detection
(
	# shellcheck disable=SC2329
	uname() { echo "MINGW64_NT-10.0"; }
	os="$(detect_os)"
	distro="$(detect_distro)"

	assert_equals "windows_msys" "$os" "Windows MSYS emulation: detect_os" &&
		assert_equals "windows" "$distro" "Windows MSYS emulation: detect_distro"
) && pass "Windows MSYS2 / Git Bash emulation detected correctly"

# Test 7: Hardware Resource Metrics
free_mb="$(get_free_disk_space_mb "$HOME")"
if ((free_mb > 0)); then
	pass "get_free_disk_space_mb returned positive integer: ${free_mb} MB"
else
	fail "get_free_disk_space_mb returned invalid value: $free_mb"
fi

ram_mb="$(get_total_ram_mb)"
if ((ram_mb > 0)); then
	pass "get_total_ram_mb returned positive integer: ${ram_mb} MB"
else
	fail "get_total_ram_mb returned invalid value: $ram_mb"
fi
