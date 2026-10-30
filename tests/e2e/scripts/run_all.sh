#!/usr/bin/env bash
# ==============================================================================
# tests/e2e/scripts/run_all.sh
# Master E2E Test Suite Orchestrator
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

BOLD="\033[1m"
GREEN="\033[0;32m"
RED="\033[0;31m"
CYAN="\033[0;36m"
YELLOW="\033[0;33m"
RESET="\033[0m"

echo -e "${BOLD}==============================================================================${RESET}"
echo -e "${BOLD}  ANX Platform — Full Docker E2E Testing Suite${RESET}"
echo -e "${BOLD}==============================================================================${RESET}"
echo -e "Platform Environment: $(uname -s) $(uname -m) (User: $(whoami))"
echo ""

TESTS=(
	"test_github_release.sh"
	"test_from_sources.sh"
	"test_curl_install.sh"
	"test_cli_and_tui.sh"
	"test_concurrency.sh"
	"test_config_and_tools.sh"
)

PASSED=0
FAILED=0
RESULTS=()

for test_script in "${TESTS[@]}"; do
	full_path="${SCRIPT_DIR}/${test_script}"
	echo -e "${CYAN}▶ Running: ${test_script}...${RESET}"
	start_time=$(date +%s)

	if bash "${full_path}"; then
		end_time=$(date +%s)
		duration=$((end_time - start_time))
		PASSED=$((PASSED + 1))
		RESULTS+=("${GREEN}PASS${RESET}  ${test_script} (${duration}s)")
	else
		end_time=$(date +%s)
		duration=$((end_time - start_time))
		FAILED=$((FAILED + 1))
		RESULTS+=("${RED}FAIL${RESET}  ${test_script} (${duration}s)")
	fi
	echo ""
done

echo -e "${BOLD}==============================================================================${RESET}"
echo -e "${BOLD}  E2E Test Execution Summary${RESET}"
echo -e "${BOLD}==============================================================================${RESET}"
for res in "${RESULTS[@]}"; do
	echo -e "  ${res}"
done
echo ""

if [[ "${FAILED}" -gt 0 ]]; then
	echo -e "${RED}FAILED: ${FAILED} of ${#TESTS[@]} test scenarios failed.${RESET}" >&2
	exit 1
else
	echo -e "${GREEN}SUCCESS: All ${PASSED} test scenarios completed cleanly!${RESET}"
	exit 0
fi
