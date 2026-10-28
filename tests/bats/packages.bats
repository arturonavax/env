#!/usr/bin/env bats
# ==============================================================================
# tests/bats/packages.bats
# Bats specification for multi-platform package translation matrix.
# ==============================================================================

setup() {
	REPO_ROOT="$(cd -- "$BATS_TEST_DIRNAME/../.." && pwd)"
	# shellcheck source=/dev/null
	source "$REPO_ROOT/src/lib/packages.sh"
}

@test "Debian translation for build-essential and ssl-dev" {
	run _translate_debian "build-essential"
	[ "$status" -eq 0 ]
	[ "$output" = "build-essential" ]

	run _translate_debian "ssl-dev"
	[ "$status" -eq 0 ]
	[ "$output" = "libssl-dev" ]
}

@test "Arch translation for build-essential and python3" {
	run _translate_arch "build-essential"
	[ "$status" -eq 0 ]
	[ "$output" = "base-devel" ]

	run _translate_arch "python3"
	[ "$status" -eq 0 ]
	[ "$output" = "python" ]
}

@test "macOS translation skips Linux-only packages" {
	run _translate_darwin "keyd"
	[ "$status" -eq 0 ]
	[ "$output" = "" ]

	run _translate_darwin "wmctrl"
	[ "$status" -eq 0 ]
	[ "$output" = "" ]
}

@test "Android Termux translation maps gcc to clang" {
	run _translate_termux "gcc"
	[ "$status" -eq 0 ]
	[ "$output" = "clang" ]
}
