#!/usr/bin/env bats
# ==============================================================================
# tests/bats/platform.bats
# Bats specification for platform discovery and Termux emulation.
# ==============================================================================

setup() {
	REPO_ROOT="$(cd -- "$BATS_TEST_DIRNAME/../.." && pwd)"
	# shellcheck source=/dev/null
	source "$REPO_ROOT/src/lib/platform.sh"
}

@test "detect_os returns a supported operating system" {
	run detect_os
	[ "$status" -eq 0 ]
	[[ "$output" =~ ^(linux|darwin|wsl|android|windows_msys|windows_cygwin)$ ]]
}

@test "detect_arch returns a standard hardware architecture" {
	run detect_arch
	[ "$status" -eq 0 ]
	[[ "$output" =~ ^(x86_64|arm64|armv7|x86_32)$ ]]
}

@test "detect_pkg_mgr detects supported package manager" {
	run detect_pkg_mgr
	[ "$status" -eq 0 ]
	[[ "$output" =~ ^(apt|dnf|yum|pacman|zypper|apk|brew|termux)$ ]]
}

@test "detect_os identifies Android Termux environment" {
	export TERMUX_VERSION="0.118.0"
	export PREFIX="/data/data/com.termux/files/usr"
	run detect_os
	[ "$status" -eq 0 ]
	[ "$output" = "android" ]
}
