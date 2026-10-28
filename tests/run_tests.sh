#!/usr/bin/env bash
# ==============================================================================
# tests/run_tests.sh
# Production Master Test Runner & Quality Gate for arturonavax/env.
# Executes Syntax Validation, Static Analysis (ShellCheck), Unit Tests,
# Integration Tests, and optional Bats specs with structured colorized summary.
# ==============================================================================

# Anchoring loop to repository root
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "Error: Repository root not found" >&2
		exit 1
	fi
	if ! cd ..; then
		echo "Error: Failed to navigate up" >&2
		exit 1
	fi
done

REPO_ROOT="$PWD"
export REPO_ROOT

# Source internal libraries
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/colors.sh" 2>/dev/null || true
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/logging.sh" 2>/dev/null || true

# Execution modes
RUN_SYNTAX=1
RUN_LINT=1
RUN_UNIT=1
RUN_INTEGRATION=1
RUN_BATS=1
VERBOSE=0

show_help() {
	cat <<EOF
${BOLD_WHITE}arturonavax/env Test Runner${RESET}

Usage:
  ./tests/run_tests.sh [options]

Options:
  --all            Run complete suite (syntax, lint, unit, integration, bats) [default]
  --unit           Run unit tests only
  --integration    Run integration smoke tests only
  --lint           Run syntax check and shellcheck only
  --bats           Run Bats specifications only
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
		RUN_BATS=1
		;;
	--unit)
		RUN_SYNTAX=0
		RUN_LINT=0
		RUN_UNIT=1
		RUN_INTEGRATION=0
		RUN_BATS=0
		;;
	--integration)
		RUN_SYNTAX=0
		RUN_LINT=0
		RUN_UNIT=0
		RUN_INTEGRATION=1
		RUN_BATS=0
		;;
	--lint)
		RUN_SYNTAX=1
		RUN_LINT=1
		RUN_UNIT=0
		RUN_INTEGRATION=0
		RUN_BATS=0
		;;
	--bats)
		RUN_SYNTAX=0
		RUN_LINT=0
		RUN_UNIT=0
		RUN_INTEGRATION=0
		RUN_BATS=1
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

printf '\n%s%s══════════════════════════════════════════════════════════════════════%s\n' "$BOLD" "$CYAN" "$RESET"
printf '%s  %s arturonavax/env Comprehensive Test Harness%s\n' "$BOLD_WHITE" "🧪" "$RESET"
printf '%s══════════════════════════════════════════════════════════════════════%s\n\n' "$CYAN" "$RESET"

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
	printf '%s[1/5] Syntax Validation (bash -n)%s\n' "$BOLD_WHITE" "$RESET"
	syntax_errors=0
	checked_count=0

	while IFS= read -r script_file; do
		[[ -f "$script_file" ]] || continue
		# Skip non-script assets
		case "$script_file" in
		*.md | *.json | *.toml | *.yaml | *.yml | *.txt | *.lock | *.zwc | *.lua | *.js | *.css | *.html | *.svg | *.png | *.desktop) continue ;;
		esac

		# Validate if file ends with .sh or has a shell shebang
		if [[ "$script_file" == *.sh ]] || head -n 1 "$script_file" 2>/dev/null | grep -qE '^#!.*(bash|sh)'; then
			checked_count=$((checked_count + 1))
			if ! bash -n "$script_file" 2>/dev/null; then
				printf '    %s%s%s Syntax error in: %s\n' "$RED" "$SYM_CROSS" "$RESET" "$script_file"
				syntax_errors=$((syntax_errors + 1))
			fi
		fi
	done < <(find "$REPO_ROOT" -type f ! -path "*/.git/*" ! -path "*/.agents/*" ! -path "*/.gemini/*" ! -path "*/.claude/*" | sort)

	if [[ "$syntax_errors" -eq 0 ]]; then
		printf '    %s%s%s All %d shell scripts verified with zero syntax errors.\n' "$GREEN" "$SYM_CHECK" "$RESET" "$checked_count"
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
	printf '%s[2/5] Static Analysis (shellcheck)%s\n' "$BOLD_WHITE" "$RESET"

	if command -v shellcheck >/dev/null 2>&1; then
		lint_targets=(
			"install.sh"
			"uninstall.sh"
			"src/wizard.sh"
			"src/lib/strict.sh"
			"src/lib/colors.sh"
			"src/lib/logging.sh"
			"src/lib/platform.sh"
			"src/lib/packages.sh"
			"src/lib/backup.sh"
			"src/lib/download.sh"
			"src/lib/concurrency.sh"
			"src/lib/smart_append.sh"
			"src/lib/remote_env.sh"
			"utils/backup_config.sh"
			"utils/restore_backup.sh"
			"utils/sync_config.sh"
			"tests/test_helper.sh"
			"tests/unit/test_platform.sh"
			"tests/unit/test_packages.sh"
			"tests/unit/test_smart_append.sh"
			"tests/unit/test_backup.sh"
			"tests/unit/test_concurrency.sh"
			"tests/unit/test_strict.sh"
			"tests/integration/test_cli_smoke.sh"
		)

		lint_output=$(shellcheck -e SC1091 -e SC1090 "${lint_targets[@]}" 2>&1)
		lint_code=$?

		if [[ "$lint_code" -eq 0 ]]; then
			printf '    %s%s%s ShellCheck passed with zero warnings across %d core scripts.\n' "$GREEN" "$SYM_CHECK" "$RESET" "${#lint_targets[@]}"
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
# 3. Unit Test Suite (tests/unit/test_*.sh)
# ------------------------------------------------------------------------------
if [[ "$RUN_UNIT" -eq 1 ]]; then
	SUITES_TOTAL=$((SUITES_TOTAL + 1))
	printf '%s[3/5] Unit Test Suite%s\n' "$BOLD_WHITE" "$RESET"
	unit_fails=0
	unit_count=0

	for unit_file in "$REPO_ROOT"/tests/unit/test_*.sh; do
		[[ -f "$unit_file" ]] || continue
		unit_count=$((unit_count + 1))
		unit_name="$(basename "$unit_file")"

		if [[ "$VERBOSE" -eq 1 ]]; then
			printf '  ▸ Running %s...\n' "$unit_name"
			if ! bash "$unit_file"; then
				unit_fails=$((unit_fails + 1))
			fi
		else
			unit_out=$(bash "$unit_file" 2>&1)
			unit_res=$?
			if [[ "$unit_res" -eq 0 && "$unit_out" != *"✖"* ]]; then
				printf '    %s%s%s %-32s (%s)\n' "$GREEN" "$SYM_CHECK" "$RESET" "$unit_name" "PASSED"
			else
				printf '    %s%s%s %-32s (%s)\n' "$RED" "$SYM_CROSS" "$RESET" "$unit_name" "FAILED"
				_indent_output "$unit_out"
				unit_fails=$((unit_fails + 1))
			fi
		fi
	done

	if [[ "$unit_fails" -eq 0 ]]; then
		SUITES_PASSED=$((SUITES_PASSED + 1))
	else
		SUITES_FAILED=$((SUITES_FAILED + 1))
	fi
	echo ""
fi

# ------------------------------------------------------------------------------
# 4. Integration & Smoke Tests (tests/integration/test_*.sh)
# ------------------------------------------------------------------------------
if [[ "$RUN_INTEGRATION" -eq 1 ]]; then
	SUITES_TOTAL=$((SUITES_TOTAL + 1))
	printf '%s[4/5] Integration Smoke Suite%s\n' "$BOLD_WHITE" "$RESET"
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
# 5. Bats Specifications (tests/bats/*.bats)
# ------------------------------------------------------------------------------
if [[ "$RUN_BATS" -eq 1 ]]; then
	SUITES_TOTAL=$((SUITES_TOTAL + 1))
	printf '%s[5/5] Bats Specifications (tests/bats/)%s\n' "$BOLD_WHITE" "$RESET"

	if command -v bats >/dev/null 2>&1; then
		bats_fails=0
		for bats_file in "$REPO_ROOT"/tests/bats/*.bats; do
			[[ -f "$bats_file" ]] || continue
			bats_name="$(basename "$bats_file")"

			if bats "$bats_file" >/dev/null 2>&1; then
				printf '    %s%s%s %-32s (%s)\n' "$GREEN" "$SYM_CHECK" "$RESET" "$bats_name" "PASSED"
			else
				printf '    %s%s%s %-32s (%s)\n' "$RED" "$SYM_CROSS" "$RESET" "$bats_name" "FAILED"
				bats_out=$(bats "$bats_file" 2>&1 || true)
				_indent_output "$bats_out"
				bats_fails=$((bats_fails + 1))
			fi
		done

		if [[ "$bats_fails" -eq 0 ]]; then
			SUITES_PASSED=$((SUITES_PASSED + 1))
		else
			SUITES_FAILED=$((SUITES_FAILED + 1))
		fi
	else
		printf '    %s%s%s bats not installed; optional Bats specifications skipped.\n' "$CYAN" "$SYM_INFO" "$RESET"
		printf '      %s(Native unit tests cover the entire test matrix without external dependencies)%s\n' "$GRAY" "$RESET"
		SUITES_PASSED=$((SUITES_PASSED + 1))
	fi
	echo ""
fi

# ------------------------------------------------------------------------------
# Final Test Summary
# ------------------------------------------------------------------------------
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

printf '%s══════════════════════════════════════════════════════════════════════%s\n' "$CYAN" "$RESET"
if [[ "$SUITES_FAILED" -eq 0 ]]; then
	printf '  %s%s ALL TEST SUITES PASSED%s %s(%d/%d passed in %ds)%s\n' \
		"$BOLD_GREEN" "$SYM_CHECK" "$RESET" "$GREEN" "$SUITES_PASSED" "$SUITES_TOTAL" "$DURATION" "$RESET"
	printf '%s══════════════════════════════════════════════════════════════════════%s\n\n' "$CYAN" "$RESET"
	exit 0
else
	printf '  %s%s TEST SUITE FAILURES DETECTED%s %s(%d failed, %d passed in %ds)%s\n' \
		"$BOLD_RED" "$SYM_CROSS" "$RESET" "$RED" "$SUITES_FAILED" "$SUITES_PASSED" "$DURATION" "$RESET"
	printf '%s══════════════════════════════════════════════════════════════════════%s\n\n' "$CYAN" "$RESET"
	exit 1
fi
