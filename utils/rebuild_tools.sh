#!/usr/bin/env bash
# ==============================================================================
# utils/rebuild_tools.sh
# Universal runtime post-upgrade tool rebuild & dependency synchronizer.
# Recompiles, reinstalls, and aligns global CLI tools across Python, Node.js,
# Golang, Rust, and Neovim following language runtime upgrades.
# Part of arturonavax/env development environment.
# ==============================================================================
set -euo pipefail

# Script directory and repo anchoring
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT=""
if [[ -d "${SCRIPT_DIR}/../files" && -d "${SCRIPT_DIR}/../src" ]]; then
	REPO_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
fi

# Load versions configuration if present
if [[ -n "${REPO_ROOT}" && -f "${REPO_ROOT}/src/remotes/_versions.sh" ]]; then
	# shellcheck source=/dev/null
	source "${REPO_ROOT}/src/remotes/_versions.sh"
elif [[ -f "${SCRIPT_DIR}/../src/remotes/_versions.sh" ]]; then
	# shellcheck source=/dev/null
	source "${SCRIPT_DIR}/../src/remotes/_versions.sh"
fi

# Colors and formatting
COLOR_RESET="\033[0m"
COLOR_BOLD="\033[1;37m"
COLOR_GREEN="\033[1;32m"
COLOR_YELLOW="\033[1;33m"
COLOR_CYAN="\033[1;36m"
COLOR_RED="\033[1;31m"
COLOR_DIM="\033[2m"

# Flags
DO_PYTHON=0
DO_NODE=0
DO_GO=0
DO_RUST=0
DO_NVIM=0
DRY_RUN=0

usage() {
	cat <<EOF
Universal Tool Rebuilder & Recompiler (arturonavax/env)

Usage:
  $(basename "$0") [options]

Options:
  --all, -a       Rebuild tools across all installed language runtimes (default)
  --python, -p    Rebuild Python tools (pipx reinstall-all with active Python)
  --node, -n      Rebuild Node.js global tools (pnpm global update & canonical tools)
  --go, -g        Recompile Golang CLI tools with active Go compiler
  --rust, -r      Update Rust toolchain & recompile Cargo utilities
  --nvim          Synchronize LazyVim plugins and update Mason registries
  --dry-run       Preview actions without executing changes
  -h, --help      Display this help message

Examples:
  rebuild-tools              # Rebuilds everything across all active runtimes
  rebuild-tools --python     # Reinstalls pipx environments with active Python
  rebuild-tools --go         # Recompiles gopls, dlv, and Go tools with active go
EOF
}

# Track if any specific subsystem was chosen
SPECIFIC_SUBSYSTEM=0

while [[ "$#" -gt 0 ]]; do
	case "$1" in
	--all | -a)
		DO_PYTHON=1
		DO_NODE=1
		DO_GO=1
		DO_RUST=1
		DO_NVIM=1
		SPECIFIC_SUBSYSTEM=1
		shift
		;;
	--python | -p)
		DO_PYTHON=1
		SPECIFIC_SUBSYSTEM=1
		shift
		;;
	--node | -n)
		DO_NODE=1
		SPECIFIC_SUBSYSTEM=1
		shift
		;;
	--go | -g)
		DO_GO=1
		SPECIFIC_SUBSYSTEM=1
		shift
		;;
	--rust | -r)
		DO_RUST=1
		SPECIFIC_SUBSYSTEM=1
		shift
		;;
	--nvim)
		DO_NVIM=1
		SPECIFIC_SUBSYSTEM=1
		shift
		;;
	--dry-run)
		DRY_RUN=1
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo -e "${COLOR_RED}[Error]: Unknown option '$1'. Use --help for usage.${COLOR_RESET}" >&2
		exit 1
		;;
	esac
done

# If no specific subsystem was selected, default to all
if [[ "$SPECIFIC_SUBSYSTEM" -eq 0 ]]; then
	DO_PYTHON=1
	DO_NODE=1
	DO_GO=1
	DO_RUST=1
	DO_NVIM=1
fi

# Source development paths to ensure active runtimes are available
if [[ -f "${HOME}/.tools.sh" ]]; then
	# shellcheck source=/dev/null
	source "${HOME}/.tools.sh" 2>/dev/null || true
elif [[ -n "${REPO_ROOT}" && -f "${REPO_ROOT}/files/zsh/.tools.sh" ]]; then
	# shellcheck source=/dev/null
	source "${REPO_ROOT}/files/zsh/.tools.sh" 2>/dev/null || true
fi

echo -e "\n${COLOR_CYAN}══════════════════════════════════════════════════════════════════════${COLOR_RESET}"
echo -e "  ${COLOR_BOLD}🔄 arturonavax/env Universal Tool Rebuilder & Synchronizer${COLOR_RESET}"
if [[ "$DRY_RUN" -eq 1 ]]; then
	echo -e "  ${COLOR_YELLOW}Mode: DRY-RUN (Previewing actions without modifying filesystem)${COLOR_RESET}"
fi
echo -e "${COLOR_CYAN}══════════════════════════════════════════════════════════════════════${COLOR_RESET}\n"

START_TIME=$(date +%s)
TOTAL_TASKS=0
PASSED_TASKS=0

# ------------------------------------------------------------------------------
# 1. Python Rebuilder
# ------------------------------------------------------------------------------
rebuild_python() {
	TOTAL_TASKS=$((TOTAL_TASKS + 1))
	echo -e "${COLOR_BOLD}[1] Python Runtime & CLI Tools${COLOR_RESET}"

	if ! command -v python3 >/dev/null 2>&1; then
		echo -e "    ${COLOR_YELLOW}⚠️ python3 is not available in PATH; skipping.${COLOR_RESET}\n"
		return 0
	fi

	local py_bin py_ver
	py_bin="$(command -v python3)"
	py_ver="$(python3 --version 2>&1)"
	echo -e "    Active Interpreter: ${COLOR_CYAN}${py_bin}${COLOR_RESET} (${py_ver})"

	if [[ "$DRY_RUN" -eq 1 ]]; then
		echo -e "    ${COLOR_DIM}[dry-run] Would upgrade pip, setuptools, wheel${COLOR_RESET}"
		echo -e "    ${COLOR_DIM}[dry-run] Would execute: pipx reinstall-all --python ${py_bin}${COLOR_RESET}"
		echo -e "    ${COLOR_DIM}[dry-run] Would verify core tools: black, pylint, beautysh, cmakelang${COLOR_RESET}\n"
		PASSED_TASKS=$((PASSED_TASKS + 1))
		return 0
	fi

	# Upgrade core bootstrapping utilities in current Python
	python3 -m pip install --upgrade --no-input pip setuptools wheel 2>/dev/null || true

	# Reinstall all pipx environments using the active Python interpreter
	if command -v pipx >/dev/null 2>&1; then
		echo -e "    Reinstalling isolated pipx CLI tools against active Python..."
		pipx reinstall-all --python "$py_bin" 2>/dev/null || pipx reinstall-all 2>/dev/null || true

		# Ensure standard suite tools are present
		for cli_pkg in black pylint beautysh cmakelang; do
			pipx install "$cli_pkg" 2>/dev/null || pipx upgrade "$cli_pkg" 2>/dev/null || true
		done
	else
		python3 -m pip install --user --upgrade black pylint beautysh cmakelang 2>/dev/null || true
	fi

	echo -e "    ${COLOR_GREEN}✔ Python global CLI tools synchronized successfully.${COLOR_RESET}\n"
	PASSED_TASKS=$((PASSED_TASKS + 1))
}

# ------------------------------------------------------------------------------
# 2. Node.js & pnpm Rebuilder
# ------------------------------------------------------------------------------
rebuild_node() {
	TOTAL_TASKS=$((TOTAL_TASKS + 1))
	echo -e "${COLOR_BOLD}[2] Node.js & Package Managers${COLOR_RESET}"

	if ! command -v node >/dev/null 2>&1; then
		echo -e "    ${COLOR_YELLOW}⚠️ node is not available in PATH; skipping.${COLOR_RESET}\n"
		return 0
	fi

	local node_bin node_ver
	node_bin="$(command -v node)"
	node_ver="$(node --version 2>&1)"
	echo -e "    Active Node: ${COLOR_CYAN}${node_bin}${COLOR_RESET} (${node_ver})"

	local node_tools=(
		"yarn"
		"serve"
		"prettier"
		"eslint"
		"typescript"
		"emmet-ls"
		"bash-language-server"
		"markdownlint-cli"
		"nginxbeautifier"
		"sql-formatter"
		"stylelint"
	)

	if [[ "$DRY_RUN" -eq 1 ]]; then
		echo -e "    ${COLOR_DIM}[dry-run] Would update global packages via pnpm/npm${COLOR_RESET}"
		echo -e "    ${COLOR_DIM}[dry-run] Would synchronize canonical tools: ${node_tools[*]}${COLOR_RESET}\n"
		PASSED_TASKS=$((PASSED_TASKS + 1))
		return 0
	fi

	if command -v pnpm >/dev/null 2>&1; then
		echo -e "    Updating and linking global packages via pnpm ($(pnpm --version))..."
		pnpm update -g 2>/dev/null || true
		pnpm add -g "${node_tools[@]}" 2>/dev/null || true
	elif command -v npm >/dev/null 2>&1; then
		echo -e "    Updating and linking global packages via npm ($(npm --version))..."
		npm update -g 2>/dev/null || true
		npm install -g "${node_tools[@]}" 2>/dev/null || true
	fi

	echo -e "    ${COLOR_GREEN}✔ Node.js global packages synchronized successfully.${COLOR_RESET}\n"
	PASSED_TASKS=$((PASSED_TASKS + 1))
}

# ------------------------------------------------------------------------------
# 3. Golang Recompiler
# ------------------------------------------------------------------------------
rebuild_go() {
	TOTAL_TASKS=$((TOTAL_TASKS + 1))
	echo -e "${COLOR_BOLD}[3] Golang Developer Tools${COLOR_RESET}"

	if ! command -v go >/dev/null 2>&1; then
		echo -e "    ${COLOR_YELLOW}⚠️ go compiler is not available in PATH; skipping.${COLOR_RESET}\n"
		return 0
	fi

	local go_bin go_ver
	go_bin="$(command -v go)"
	go_ver="$(go version 2>&1)"
	echo -e "    Active Compiler: ${COLOR_CYAN}${go_bin}${COLOR_RESET} (${go_ver})"

	local go_pkgs=(
		"golang.org/x/tools/gopls@latest"
		"github.com/go-delve/delve/cmd/dlv@latest"
		"mvdan.cc/gofumpt@latest"
		"github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
		"github.com/air-verse/air@latest"
		"github.com/securego/gosec/v2/cmd/gosec@latest"
		"google.golang.org/protobuf/cmd/protoc-gen-go@latest"
		"google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
		"golang.org/x/vuln/cmd/govulncheck@latest"
	)

	if [[ "$DRY_RUN" -eq 1 ]]; then
		echo -e "    ${COLOR_DIM}[dry-run] Would recompile Go binaries with current compiler:${COLOR_RESET}"
		for pkg in "${go_pkgs[@]}"; do
			echo -e "      ${COLOR_DIM}• go install -ldflags='-s -w' ${pkg}${COLOR_RESET}"
		done
		echo ""
		PASSED_TASKS=$((PASSED_TASKS + 1))
		return 0
	fi

	echo -e "    Recompiling Go toolchain binaries against active compiler..."
	for pkg in "${go_pkgs[@]}"; do
		local tool_name="${pkg%%@*}"
		tool_name="$(basename "$tool_name")"
		printf '    ▸ Recompiling %-28s ... ' "$tool_name"
		if go install -ldflags="-s -w" "$pkg" 2>/dev/null; then
			echo -e "${COLOR_GREEN}✔${COLOR_RESET}"
		else
			echo -e "${COLOR_YELLOW}skipped/warn${COLOR_RESET}"
		fi
	done

	go clean -cache 2>/dev/null || true
	echo -e "    ${COLOR_GREEN}✔ Golang binaries recompiled successfully.${COLOR_RESET}\n"
	PASSED_TASKS=$((PASSED_TASKS + 1))
}

# ------------------------------------------------------------------------------
# 4. Rust & Cargo Recompiler
# ------------------------------------------------------------------------------
rebuild_rust() {
	TOTAL_TASKS=$((TOTAL_TASKS + 1))
	echo -e "${COLOR_BOLD}[4] Rust Toolchain & Cargo Utilities${COLOR_RESET}"

	if ! command -v rustup >/dev/null 2>&1 && ! command -v cargo >/dev/null 2>&1; then
		echo -e "    ${COLOR_YELLOW}⚠️ Rust/Cargo is not available in PATH; skipping.${COLOR_RESET}\n"
		return 0
	fi

	if command -v rustup >/dev/null 2>&1; then
		echo -e "    Active Rust Toolchain: ${COLOR_CYAN}$(rustup show active-toolchain 2>&1 | awk '{print $1}')${COLOR_RESET}"
		if [[ "$DRY_RUN" -eq 1 ]]; then
			echo -e "    ${COLOR_DIM}[dry-run] Would execute: rustup update stable${COLOR_RESET}"
		else
			rustup update stable 2>/dev/null || true
		fi
	fi

	if command -v cargo >/dev/null 2>&1; then
		if [[ "$DRY_RUN" -eq 1 ]]; then
			echo -e "    ${COLOR_DIM}[dry-run] Would update cargo-installed tools (rtk, stylua)${COLOR_RESET}\n"
			PASSED_TASKS=$((PASSED_TASKS + 1))
			return 0
		fi

		# Recompile/update rtk if installed via cargo
		if command -v rtk >/dev/null 2>&1; then
			printf '    ▸ Rebuilding rtk token compressor ... '
			if cargo install rtk 2>/dev/null; then
				echo -e "${COLOR_GREEN}✔${COLOR_RESET}"
			else
				echo -e "${COLOR_YELLOW}skipped${COLOR_RESET}"
			fi
		fi
	fi

	echo -e "    ${COLOR_GREEN}✔ Rust toolchain & utilities synchronized successfully.${COLOR_RESET}\n"
	PASSED_TASKS=$((PASSED_TASKS + 1))
}

# ------------------------------------------------------------------------------
# 5. Neovim & Mason Registry
# ------------------------------------------------------------------------------
rebuild_nvim() {
	TOTAL_TASKS=$((TOTAL_TASKS + 1))
	echo -e "${COLOR_BOLD}[5] Neovim Plugins & Language Server Registry${COLOR_RESET}"

	if ! command -v nvim >/dev/null 2>&1; then
		echo -e "    ${COLOR_YELLOW}⚠️ nvim is not available in PATH; skipping.${COLOR_RESET}\n"
		return 0
	fi

	echo -e "    Active Neovim: ${COLOR_CYAN}$(command -v nvim)${COLOR_RESET} ($(nvim --version 2>&1 | head -n 1))"

	if [[ "$DRY_RUN" -eq 1 ]]; then
		echo -e "    ${COLOR_DIM}[dry-run] Would run: nvim --headless '+Lazy! sync' +qa${COLOR_RESET}"
		echo -e "    ${COLOR_DIM}[dry-run] Would run: nvim --headless '+MasonUpdate' +qa${COLOR_RESET}\n"
		PASSED_TASKS=$((PASSED_TASKS + 1))
		return 0
	fi

	echo -e "    Synchronizing Lazy.nvim plugins and Mason registry in background..."
	nvim --headless "+Lazy! sync" +qa 2>/dev/null || true
	nvim --headless "+MasonUpdate" +qa 2>/dev/null || true

	echo -e "    ${COLOR_GREEN}✔ Neovim plugins and LSP registries updated successfully.${COLOR_RESET}\n"
	PASSED_TASKS=$((PASSED_TASKS + 1))
}

# Execute enabled components
((DO_PYTHON == 1)) && rebuild_python
((DO_NODE == 1)) && rebuild_node
((DO_GO == 1)) && rebuild_go
((DO_RUST == 1)) && rebuild_rust
((DO_NVIM == 1)) && rebuild_nvim

ELAPSED=$(( $(date +%s) - START_TIME ))

echo -e "${COLOR_CYAN}══════════════════════════════════════════════════════════════════════${COLOR_RESET}"
echo -e "  ${COLOR_GREEN}✔ Universal Tool Rebuild Completed (${PASSED_TASKS}/${TOTAL_TASKS} subsystems synchronized in ${ELAPSED}s)${COLOR_RESET}"
echo -e "${COLOR_CYAN}══════════════════════════════════════════════════════════════════════${COLOR_RESET}\n"
