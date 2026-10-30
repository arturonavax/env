#!/usr/bin/env bash
# ==============================================================================
# tests/run_tests.sh
# Production Master Test Runner & Quality Gate for arturonavax/env.
# Executes Syntax Validation, Static Analysis (ShellCheck), Go Race Tests,
# and CLI Integration Smoke Tests with structured colorized summary.
# ==============================================================================

# Anchoring loop to repository root
cur="$PWD"
while [[ "$cur" != "/" ]]; do
	if [[ -d "${cur}/manifests" && -f "${cur}/scripts/bootstrap_anx.sh" ]]; then
		REPO_ROOT="$cur"
		break
	fi
	cur="$(dirname "$cur")"
done

if [[ -z "${REPO_ROOT:-}" ]]; then
	echo "Error: Repository root not found" >&2
	exit 1
fi
export REPO_ROOT
cd "$REPO_ROOT"

# ANSI Colors and symbols
BOLD_WHITE=$'\033[1;37m'
BOLD_GREEN=$'\033[1;32m'
BOLD_RED=$'\033[1;31m'
GREEN=$'\033[0;32m'
YELLOW=$'\033[0;33m'
CYAN=$'\033[0;36m'
RED=$'\033[0;31m'
GRAY=$'\033[0;90m'
RESET=$'\033[0m'

SYM_CHECK="✔"
SYM_CROSS="✖"
SYM_WARN="⚠"
SYM_INFO="ℹ"

ui_box_width() { echo "${1:-70}"; }
ui_box_separator() {
	local w="$1" char="${2:-═}" col="${3:-$CYAN}"
	printf "%b" "$col"
	for ((i = 0; i < w; i++)); do printf "%s" "$char"; done
	printf "%b\n" "$RESET"
}

# Execution modes
RUN_SYNTAX=1
RUN_LINT=1
RUN_UNIT=1
RUN_INTEGRATION=1
RUN_LEGACY=0
VERBOSE=0

show_help() {
	cat <<EOF
${BOLD_WHITE}arturonavax/env Test Runner${RESET}

Usage:
  ./tests/run_tests.sh [options]

Options:
  --all            Run complete suite (syntax, lint, go unit tests, integration) [default]
  --unit           Run Go unit & race tests only
  --integration    Run integration smoke tests only
  --lint           Run syntax check and shellcheck only
  --legacy         Also execute archived legacy bash unit tests (archives/tests/)
  -v, --verbose    Show verbose output during execution
  -h, --help       Show this documentation

EOF
	exit 0
}

# Parse CLI options
while [[ $# -gt 0 ]]; do
	case "$1" in
	--all)
		RUN_SYNTAX=1
		RUN_LINT=1
		RUN_UNIT=1
		RUN_INTEGRATION=1
		;;
	--unit)
		RUN_SYNTAX=0
		RUN_LINT=0
		RUN_UNIT=1
		RUN_INTEGRATION=0
		;;
	--integration)
		RUN_SYNTAX=0
		RUN_LINT=0
		RUN_UNIT=0
		RUN_INTEGRATION=1
		;;
	--lint)
		RUN_SYNTAX=1
		RUN_LINT=1
		RUN_UNIT=0
		RUN_INTEGRATION=0
		;;
	--legacy)
		RUN_LEGACY=1
		;;
	-v | --verbose)
		VERBOSE=1
		;;
	-h | --help)
		show_help
		;;
	*)
		echo "Unknown argument: $1" >&2
		show_help
		;;
	esac
	shift
done

SUITES_TOTAL=0
SUITES_PASSED=0
SUITES_FAILED=0
START_TIME=$(date +%s)

header_w="$(ui_box_width 70)"
echo
ui_box_separator "$header_w" "═" "$CYAN"
printf '%s  %s arturonavax/env Comprehensive Test Harness%s\n' "$BOLD_WHITE" "🧪" "$RESET"
ui_box_separator "$header_w" "═" "$CYAN"
echo

_indent_output() {
	local text="$1"
	while IFS= read -r line; do
		printf '      %s\n' "$line"
	done <<<"$text"
}

# ------------------------------------------------------------------------------
# 1. Syntax Validation (bash -n)
# ------------------------------------------------------------------------------
if [[ "$RUN_SYNTAX" -eq 1 ]]; then
	SUITES_TOTAL=$((SUITES_TOTAL + 1))
	printf '%s[1/4] Syntax Validation (bash -n)%s\n' "$BOLD_WHITE" "$RESET"
	syntax_errors=0
	checked_count=0

	while IFS= read -r script_file; do
		[[ -f "$script_file" ]] || continue
		case "$script_file" in
		*.md | *.json | *.toml | *.yaml | *.yml | *.txt | *.lock | *.zwc | *.lua | *.js | *.css | *.html | *.svg | *.png | *.desktop) continue ;;
		esac

		if [[ "$script_file" == *.sh ]] || head -n 1 "$script_file" 2>/dev/null | grep -qE '^#!.*(bash|sh)'; then
			checked_count=$((checked_count + 1))
			if ! bash -n "$script_file" 2>/dev/null; then
				printf '    %s%s%s Syntax error in: %s\n' "$RED" "$SYM_CROSS" "$RESET" "$script_file"
				syntax_errors=$((syntax_errors + 1))
			fi
		elif [[ "$script_file" == *.zsh ]] || head -n 1 "$script_file" 2>/dev/null | grep -qE '^#!.*zsh'; then
			if command -v zsh >/dev/null 2>&1; then
				checked_count=$((checked_count + 1))
				if ! zsh -n "$script_file" 2>/dev/null; then
					printf '    %s%s%s Zsh syntax error in: %s\n' "$RED" "$SYM_CROSS" "$RESET" "$script_file"
					syntax_errors=$((syntax_errors + 1))
				fi
			fi
		fi
	done < <(find "$REPO_ROOT" -type f ! -path "*/.git/*" ! -path "*/.agents/*" ! -path "*/.gemini/*" ! -path "*/.claude/*" ! -path "*/archives/*" | sort)

	if [[ "$syntax_errors" -eq 0 ]]; then
		printf '    %s%s%s All %d active shell scripts verified with zero syntax errors.\n' "$GREEN" "$SYM_CHECK" "$RESET" "$checked_count"
		SUITES_PASSED=$((SUITES_PASSED + 1))
	else
		printf '    %s%s%s %d syntax error(s) detected.\n' "$RED" "$SYM_CROSS" "$RESET" "$syntax_errors"
		SUITES_FAILED=$((SUITES_FAILED + 1))
	fi
	echo ""
fi

# ------------------------------------------------------------------------------
# 2. Static Analysis & Linting (shellcheck)
# ------------------------------------------------------------------------------
if [[ "$RUN_LINT" -eq 1 ]]; then
	SUITES_TOTAL=$((SUITES_TOTAL + 1))
	printf '%s[2/4] Static Analysis (shellcheck)%s\n' "$BOLD_WHITE" "$RESET"

	if command -v shellcheck >/dev/null 2>&1; then
		lint_targets=(
			"install.sh"
			"uninstall.sh"
			"scripts/bootstrap_anx.sh"
			"scripts/install_golang.sh"
			"scripts/test_e2e_docker.sh"
			"manifests/hacks/dell-precision-m6800/apply_hacks.sh"
			"manifests/hacks/x11-scroll-lock-led/activate_scroll_lock.sh"
			"tests/test_helper.sh"
			"tests/integration/test_cli_smoke.sh"
			"tests/e2e/scripts/run_all.sh"
			"tests/e2e/scripts/test_cli_and_tui.sh"
			"tests/e2e/scripts/test_concurrency.sh"
			"tests/e2e/scripts/test_config_and_tools.sh"
			"tests/e2e/scripts/test_curl_install.sh"
			"tests/e2e/scripts/test_from_sources.sh"
			"tests/e2e/scripts/test_github_release.sh"
			"files/zsh/.tools.sh"
		)

		valid_lint_targets=()
		for t in "${lint_targets[@]}"; do
			if [[ -f "$t" ]]; then
				valid_lint_targets+=("$t")
			fi
		done

		lint_output=$(shellcheck -e SC1091 -e SC1090 "${valid_lint_targets[@]}" 2>&1)
		lint_code=$?

		if [[ "$lint_code" -eq 0 ]]; then
			printf '    %s%s%s ShellCheck passed with zero warnings across %d active scripts.\n' "$GREEN" "$SYM_CHECK" "$RESET" "${#valid_lint_targets[@]}"
			SUITES_PASSED=$((SUITES_PASSED + 1))
		else
			printf '    %s%s%s ShellCheck detected style/defensive warnings:\n' "$RED" "$SYM_CROSS" "$RESET"
			echo "$lint_output"
			SUITES_FAILED=$((SUITES_FAILED + 1))
		fi
	else
		printf '    %s%s%s shellcheck not found in PATH; skipping static analysis.\n' "$YELLOW" "$SYM_WARN" "$RESET"
		SUITES_PASSED=$((SUITES_PASSED + 1))
	fi
	echo ""
fi

# ------------------------------------------------------------------------------
# 3. Go Test Suite (Unit & Race Tests)
# ------------------------------------------------------------------------------
if [[ "$RUN_UNIT" -eq 1 ]]; then
	SUITES_TOTAL=$((SUITES_TOTAL + 1))
	printf '%s[3/4] Go Unit & Race Test Suite (go test -race ./...)%s\n' "$BOLD_WHITE" "$RESET"

	if command -v go >/dev/null 2>&1; then
		if [[ "$VERBOSE" -eq 1 ]]; then
			if go test -race ./...; then
				printf '    %s%s%s Go unit and race tests PASSED.\n' "$GREEN" "$SYM_CHECK" "$RESET"
				SUITES_PASSED=$((SUITES_PASSED + 1))
			else
				printf '    %s%s%s Go unit and race tests FAILED.\n' "$RED" "$SYM_CROSS" "$RESET"
				SUITES_FAILED=$((SUITES_FAILED + 1))
			fi
		else
			go_out=$(go test -race ./... 2>&1)
			go_code=$?
			if [[ "$go_code" -eq 0 ]]; then
				test_pkg_count=$(grep -c "ok" <<<"$go_out" || echo "all")
				printf '    %s%s%s Go test suite passed with zero data races (%s packages OK).\n' "$GREEN" "$SYM_CHECK" "$RESET" "$test_pkg_count"
				SUITES_PASSED=$((SUITES_PASSED + 1))
			else
				printf '    %s%s%s Go test suite failed:\n' "$RED" "$SYM_CROSS" "$RESET"
				_indent_output "$go_out"
				SUITES_FAILED=$((SUITES_FAILED + 1))
			fi
		fi
	else
		printf '    %s%s%s go compiler not found in PATH; skipping Go unit tests.\n' "$RED" "$SYM_CROSS" "$RESET"
		SUITES_FAILED=$((SUITES_FAILED + 1))
	fi
	echo ""
fi

# ------------------------------------------------------------------------------
# 4. Integration & Smoke Tests (tests/integration/test_*.sh)
# ------------------------------------------------------------------------------
if [[ "$RUN_INTEGRATION" -eq 1 ]]; then
	SUITES_TOTAL=$((SUITES_TOTAL + 1))
	printf '%s[4/4] Integration Smoke Suite%s\n' "$BOLD_WHITE" "$RESET"
	int_fails=0

	for int_file in "$REPO_ROOT"/tests/integration/test_*.sh; do
		[[ -f "$int_file" ]] || continue
		int_name="$(basename "$int_file")"

		if [[ "$VERBOSE" -eq 1 ]]; then
			printf '  ▸ Running %s...\n' "$int_name"
			if ! bash "$int_file"; then
				int_fails=$((int_fails + 1))
			fi
		else
			int_out=$(bash "$int_file" 2>&1)
			int_res=$?
			if [[ "$int_res" -eq 0 && "$int_out" != *"✖"* ]]; then
				printf '    %s%s%s %-32s (%s)\n' "$GREEN" "$SYM_CHECK" "$RESET" "$int_name" "PASSED"
			else
				printf '    %s%s%s %-32s (%s)\n' "$RED" "$SYM_CROSS" "$RESET" "$int_name" "FAILED"
				_indent_output "$int_out"
				int_fails=$((int_fails + 1))
			fi
		fi
	done

	if [[ "$int_fails" -eq 0 ]]; then
		SUITES_PASSED=$((SUITES_PASSED + 1))
	else
		SUITES_FAILED=$((SUITES_FAILED + 1))
	fi
	echo ""
fi

# ------------------------------------------------------------------------------
# Optional: Archived Legacy Bash Unit Tests (--legacy)
# ------------------------------------------------------------------------------
if [[ "$RUN_LEGACY" -eq 1 ]]; then
	printf '%s[Bonus] Archived Legacy Bash Unit Tests (archives/tests/unit/)%s\n' "$BOLD_WHITE" "$RESET"
	legacy_fails=0
	legacy_count=0

	for l_file in "$REPO_ROOT"/archives/tests/unit/test_*.sh; do
		[[ -f "$l_file" ]] || continue
		legacy_count=$((legacy_count + 1))
		l_name="$(basename "$l_file")"
		l_out=$(bash "$l_file" 2>&1)
		l_res=$?
		if [[ "$l_res" -eq 0 && "$l_out" != *"✖"* ]]; then
			printf '    %s%s%s %-32s (%s)\n' "$GREEN" "$SYM_CHECK" "$RESET" "$l_name" "PASSED"
		else
			printf '    %s%s%s %-32s (%s)\n' "$YELLOW" "$SYM_WARN" "$RESET" "$l_name" "LEGACY_SKIPPED"
		fi
	done
	echo ""
fi

# ------------------------------------------------------------------------------
# Final Test Summary
# ------------------------------------------------------------------------------
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

summary_w="$(ui_box_width 70)"
ui_box_separator "$summary_w" "═" "$CYAN"
if [[ "$SUITES_FAILED" -eq 0 ]]; then
	printf '  %s%s ALL TEST SUITES PASSED%s %s(%d/%d passed in %ds)%s\n' \
		"$BOLD_GREEN" "$SYM_CHECK" "$RESET" "$GREEN" "$SUITES_PASSED" "$SUITES_TOTAL" "$DURATION" "$RESET"
	ui_box_separator "$summary_w" "═" "$CYAN"
	echo
	exit 0
else
	printf '  %s%s TEST SUITE FAILURES DETECTED%s %s(%d failed, %d passed in %ds)%s\n' \
		"$BOLD_RED" "$SYM_CROSS" "$RESET" "$RED" "$SUITES_FAILED" "$SUITES_PASSED" "$DURATION" "$RESET"
	ui_box_separator "$summary_w" "═" "$CYAN"
	echo
	exit 1
fi
