#!/usr/bin/env bash
# ==============================================================================
# scripts/test_e2e_docker.sh
# Multi-platform Docker E2E Test Suite Runner for ANX Platform.
#
# Usage:
#   ./scripts/test_e2e_docker.sh [all|debian|alpine]
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${REPO_ROOT}/tests/e2e/docker-compose.yml"

TARGET="${1:-debian}"

BOLD="\033[1m"
CYAN="\033[0;36m"
GREEN="\033[0;32m"
RED="\033[0;31m"
RESET="\033[0m"

echo -e "${BOLD}==============================================================================${RESET}"
echo -e "${BOLD}  ANX Platform — Docker E2E Test Suite Runner${RESET}"
echo -e "${BOLD}==============================================================================${RESET}"
echo -e "Target Suite: ${CYAN}${TARGET}${RESET}"
echo ""

if ! command -v docker >/dev/null 2>&1; then
	echo -e "${RED}[ERROR]: 'docker' command not found. Docker is required to run E2E tests.${RESET}" >&2
	exit 1
fi

cleanup() {
	echo ""
	echo -e "  [INFO] Tearing down E2E test containers..."
	docker compose -f "${COMPOSE_FILE}" down --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

case "${TARGET}" in
	debian)
		echo -e "  [INFO] Building and starting mock server + Debian test harness..."
		docker compose -f "${COMPOSE_FILE}" up --build --abort-on-container-exit --exit-code-from test-debian test-debian
		;;
	alpine)
		echo -e "  [INFO] Building and starting mock server + Alpine test harness..."
		docker compose -f "${COMPOSE_FILE}" up --build --abort-on-container-exit --exit-code-from test-alpine test-alpine
		;;
	all)
		echo -e "  [INFO] Running Debian E2E suite..."
		docker compose -f "${COMPOSE_FILE}" up --build --abort-on-container-exit --exit-code-from test-debian test-debian
		echo ""
		echo -e "  [INFO] Running Alpine E2E suite..."
		docker compose -f "${COMPOSE_FILE}" up --build --abort-on-container-exit --exit-code-from test-alpine test-alpine
		;;
	*)
		echo -e "${RED}[ERROR]: Unknown target: ${TARGET}. Valid options: all, debian, alpine.${RESET}" >&2
		exit 1
		;;
esac

echo ""
echo -e "${GREEN}${BOLD}✓ Docker E2E test run finished successfully!${RESET}"
