#!/usr/bin/env bash
# ==============================================================================
# src/remotes/install_python.sh
# Multi-platform Python & pyenv environment provisioner.
# Can run standalone via curl or integrated within arturonavax/env.
# ==============================================================================
set -euo pipefail

# Error trap
exit_error() {
	local exit_code=$?
	local line_no=$1
	echo -e "\n\033[1;31m[Python Installer Error]:\033[0m Command failed at line ${line_no} (exit code ${exit_code})." >&2
	exit "${exit_code}"
}
trap 'exit_error $LINENO' ERR

# Resolve colors and paths
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT=""
if [[ -d "${SCRIPT_DIR}/../../files" && -d "${SCRIPT_DIR}/../../src" ]]; then
	REPO_ROOT="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
fi

# Load versions configuration if present
if [[ -f "${SCRIPT_DIR}/_versions.sh" ]]; then
	# shellcheck source=/dev/null
	source "${SCRIPT_DIR}/_versions.sh"
elif [[ -n "${REPO_ROOT}" && -f "${REPO_ROOT}/src/remotes/_versions.sh" ]]; then
	# shellcheck source=/dev/null
	source "${REPO_ROOT}/src/remotes/_versions.sh"
fi

# Load colors if present
if [[ -f "${SCRIPT_DIR}/_vars_colors.sh" ]]; then
	# shellcheck source=/dev/null
	source "${SCRIPT_DIR}/_vars_colors.sh"
elif [[ -n "${REPO_ROOT}" && -f "${REPO_ROOT}/src/lib/colors.sh" ]]; then
	# shellcheck source=/dev/null
	source "${REPO_ROOT}/src/lib/colors.sh"
fi

COLOR_RESET="${fgcolor_reset:-\033[0m}"
COLOR_BOLD="${fgcolor_white_bold:-\033[1;37m}"
COLOR_GREEN="${fgcolor_green_bold:-\033[1;32m}"
COLOR_YELLOW="${fgcolor_yellow_bold:-\033[1;33m}"
COLOR_CYAN="${fgcolor_cyan_bold:-\033[1;36m}"
COLOR_RED="${fgcolor_red_bold:-\033[1;31m}"

# Default parameters
INSTALL_MODE=0
UPDATE_MODE=0
FORCE_REINSTALL=0
TARGET_VERSION="${PYTHON_VERSION:-3.12.9}"

# Parse flags
while getopts ':iufv:h' flag; do
	case "$flag" in
	i) INSTALL_MODE=1 ;;
	u) UPDATE_MODE=1; INSTALL_MODE=1 ;;
	f) FORCE_REINSTALL=1; INSTALL_MODE=1 ;;
	v) TARGET_VERSION="$OPTARG"; INSTALL_MODE=1 ;;
	h)
		echo "Usage: $0 [-i] [-u] [-f] [-v <version>]"
		echo "  -i             Install target Python version and configure environment"
		echo "  -u             Update pyenv repo and list available releases"
		echo "  -f             Force reinstallation even if version exists"
		echo "  -v <version>   Target specific Python version (default: ${TARGET_VERSION})"
		exit 0
		;;
	*)
		echo -e "${COLOR_RED}[Python Installer]: Unknown flag '-${OPTARG}'. Use -h for help.${COLOR_RESET}" >&2
		exit 1
		;;
	esac
done

echo -e "${COLOR_BOLD}[Python Installer]: 🐍 Target Python version: ${COLOR_CYAN}${TARGET_VERSION}${COLOR_RESET}"

# Ensure curl exists
if ! command -v curl >/dev/null 2>&1; then
	echo -e "${COLOR_RED}[Python Installer]: 'curl' is required to execute this script.${COLOR_RESET}" >&2
	exit 1
fi

# 1. macOS Command Line Tools & Homebrew bootstrap
OS="$(uname -s)"
if [[ "$OS" == "Darwin" ]]; then
	if [[ ! -e "/Library/Developer/CommandLineTools/usr/bin/git" ]]; then
		echo -e "${COLOR_BOLD}[Python Installer]: Installing macOS Command Line Tools...${COLOR_RESET}"
		xcode-select --install 2>/dev/null || true
	fi

	if ! command -v brew >/dev/null 2>&1; then
		echo -e "${COLOR_BOLD}[Python Installer]: Installing Homebrew...${COLOR_RESET}"
		/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
		brewbin="/opt/homebrew/bin/brew"
		[[ ! -f "$brewbin" ]] && brewbin="/usr/local/bin/brew"
		[[ -f "$brewbin" ]] && eval "$("$brewbin" shellenv)"
	fi
fi

# 2. Environment & PATH Setup for pyenv
export PYENV_ROOT="${PYENV_ROOT:-$HOME/.pyenv}"
if [[ ":$PATH:" != *":$PYENV_ROOT/bin:"* ]]; then
	export PATH="$PYENV_ROOT/bin:$PATH"
fi
if [[ ":$PATH:" != *":$PYENV_ROOT/shims:"* ]]; then
	export PATH="$PYENV_ROOT/shims:$PATH"
fi

# 3. Bootstrap pyenv if missing
if [[ ! -d "$PYENV_ROOT" ]]; then
	echo -e "${COLOR_BOLD}[Python Installer]: Installing pyenv (Python Version Manager)...${COLOR_RESET}"
	if [[ "$OS" == "Darwin" ]] && command -v brew >/dev/null 2>&1; then
		brew install pyenv pyenv-virtualenv || curl -fsSL https://pyenv.run | bash
	else
		curl -fsSL https://pyenv.run | bash
	fi
	echo -e "${COLOR_GREEN}[Python Installer]: ✔️ pyenv installed successfully in ${PYENV_ROOT}${COLOR_RESET}"
elif [[ "$UPDATE_MODE" -eq 1 ]]; then
	echo -e "${COLOR_BOLD}[Python Installer]: Updating pyenv repository...${COLOR_RESET}"
	if command -v pyenv >/dev/null 2>&1 && pyenv update >/dev/null 2>&1; then
		echo -e "${COLOR_GREEN}[Python Installer]: ✔️ pyenv updated successfully.${COLOR_RESET}"
	elif [[ -d "$PYENV_ROOT/.git" ]]; then
		git -C "$PYENV_ROOT" pull --quiet --ff-only 2>/dev/null || true
	fi
fi

# Initialize pyenv shims in current session
if command -v pyenv >/dev/null 2>&1; then
	eval "$(pyenv init - 2>/dev/null || true)"
fi

# 4. Check Current Status
current_version=""
if command -v python3 >/dev/null 2>&1; then
	current_version="$(python3 --version 2>&1 | awk '{print $2}')"
	if command -v python3 | grep -q ".pyenv"; then
		echo -e "${COLOR_BOLD}[Python Installer]: Active pyenv Python: ${COLOR_GREEN}${current_version}${COLOR_RESET}"
	else
		echo -e "${COLOR_YELLOW}[Python Installer]: Active Python (${current_version}) is system-managed, not managed by pyenv.${COLOR_RESET}"
	fi
fi

# If install mode wasn't explicitly requested, prompt summary and exit
if [[ "$INSTALL_MODE" -eq 0 ]]; then
	echo -e "${COLOR_BOLD}[Python Installer]: Run with ${COLOR_GREEN}'-i'${COLOR_BOLD} to provision Python ${TARGET_VERSION}.${COLOR_RESET}"
	exit 0
fi

# 5. Install System Build Dependencies for Compiling Python
echo -e "${COLOR_BOLD}[Python Installer]: Resolving development headers and compilation dependencies...${COLOR_RESET}"

if [[ -n "${REPO_ROOT}" && -f "${REPO_ROOT}/src/lib/packages.sh" ]]; then
	# Use repository unified package translation matrix
	# shellcheck source=/dev/null
	source "${REPO_ROOT}/src/lib/strict.sh" 2>/dev/null || true
	# shellcheck source=/dev/null
	source "${REPO_ROOT}/src/lib/packages.sh"
	pkg_install build-essential ssl-dev zlib-dev bz2-dev readline-dev sqlite-dev ffi-dev ncurses-dev xz-dev tk-dev xml2-dev xslt-dev
elif [[ "$OS" == "Linux" ]]; then
	if [[ -f /etc/os-release ]]; then
		# shellcheck source=/dev/null
		source /etc/os-release
		id_like="${ID_LIKE:-${ID:-}}"

		if [[ "$id_like" == *"debian"* || "$id_like" == *"ubuntu"* ]]; then
			sudo apt update -y 2>/dev/null || true
			sudo apt install -y build-essential libssl-dev zlib1g-dev libbz2-dev \
				libreadline-dev libsqlite3-dev libffi-dev libncursesw5-dev \
				xz-utils tk-dev libxml2-dev libxslt1-dev curl git
		elif [[ "$id_like" == *"fedora"* || "$id_like" == *"rhel"* || "$id_like" == *"centos"* ]]; then
			sudo dnf install -y gcc make git curl zlib-devel bzip2-devel readline-devel \
				sqlite-devel openssl-devel libffi-devel ncurses-devel xz-devel tk-devel libxml2-devel
		elif [[ "$id_like" == *"arch"* || "$ID" == "arch" ]]; then
			sudo pacman -S --needed --noconfirm base-devel openssl zlib bzip2 readline sqlite libffi ncurses xz tk curl git
		elif [[ "$id_like" == *"suse"* ]]; then
			sudo zypper in -y -t pattern devel_basis 2>/dev/null || true
			sudo zypper in -y zlib-devel libbz2-devel readline-devel sqlite3-devel libopenssl-devel libffi-devel ncurses-devel xz-devel tk-devel
		fi
	fi
elif [[ "$OS" == "Darwin" ]]; then
	brew install openssl@3 readline sqlite3 xz zlib tcl-tk curl git 2>/dev/null || true
fi

# 6. Install or Verify Target Python Version in pyenv
is_installed=0
if command -v pyenv >/dev/null 2>&1; then
	if pyenv versions --bare 2>/dev/null | grep -qx "${TARGET_VERSION}"; then
		is_installed=1
	fi
fi

if [[ "$is_installed" -eq 1 && "$FORCE_REINSTALL" -eq 0 ]]; then
	echo -e "${COLOR_GREEN}[Python Installer]: ✔️ Python ${TARGET_VERSION} is already installed in pyenv.${COLOR_RESET}"
else
	echo -e "${COLOR_BOLD}[Python Installer]: Compiling Python ${TARGET_VERSION} via pyenv (this may take 2-4 minutes)...${COLOR_RESET}"
	install_args=("-s")
	[[ "$FORCE_REINSTALL" -eq 1 ]] && install_args=("-f")

	# On macOS, export Homebrew library paths for pyenv compilation
	if [[ "$OS" == "Darwin" ]] && command -v brew >/dev/null 2>&1; then
		brew_prefix="$(brew --prefix)"
		CFLAGS="-I${brew_prefix}/include" \
		LDFLAGS="-L${brew_prefix}/lib" \
		pyenv install "${install_args[@]}" "${TARGET_VERSION}"
	else
		pyenv install "${install_args[@]}" "${TARGET_VERSION}"
	fi
	echo -e "${COLOR_GREEN}[Python Installer]: ✔️ Python ${TARGET_VERSION} compiled successfully.${COLOR_RESET}"
fi

# Set target version globally and rehash
pyenv global "${TARGET_VERSION}"
pyenv rehash
eval "$(pyenv init - 2>/dev/null || true)"

# 7. Core Python Toolchain & pipx Provisioning
echo -e "${COLOR_BOLD}[Python Installer]: Upgrading essential build utilities (pip, setuptools, wheel)...${COLOR_RESET}"
python3 -m pip install --upgrade --no-input pip setuptools wheel 2>/dev/null || true

echo -e "${COLOR_BOLD}[Python Installer]: Ensuring pipx for isolated CLI applications...${COLOR_RESET}"
if ! command -v pipx >/dev/null 2>&1; then
	python3 -m pip install --upgrade --no-input --user pipx 2>/dev/null || true
fi

if command -v pipx >/dev/null 2>&1; then
	pipx ensurepath 2>/dev/null || true
fi

echo -e "\n${COLOR_GREEN}══════════════════════════════════════════════════════════════════════${COLOR_RESET}"
echo -e "  ${COLOR_GREEN}✔ Python ${TARGET_VERSION} & pyenv configured successfully!${COLOR_RESET}"
echo -e "    Active binary: $(command -v python3 2>/dev/null || echo "$PYENV_ROOT/shims/python3")"
echo -e "    Python version: $(python3 --version 2>&1)"
echo -e "    Isolated tools: pipx available ($(command -v pipx 2>/dev/null || echo "in ~/.local/bin/pipx"))"
echo -e "${COLOR_GREEN}══════════════════════════════════════════════════════════════════════${COLOR_RESET}\n"
