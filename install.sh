#!/bin/sh
# ==============================================================================
# install.sh
# Primary POSIX-compliant multi-platform installation entry point for arturonavax/env.
# Compatible with any POSIX shell (dash, ash, bash, zsh, ksh) on Linux, macOS,
# Raspberry Pi, Alpine, Android Termux, and BSD.
#
# Supports both local repository invocation and remote piped execution:
#   curl -fsSL https://env.arturonavax.dev/install.sh | sh
#   curl -fsSL https://env.arturonavax.dev | sh
# ==============================================================================

set -eu

# If running through a pipe (e.g. curl ... | sh) and a controlling TTY exists,
# reconnect standard input so interactive wizards (Bubble Tea) can operate.
if [ ! -t 0 ] && [ -r /dev/tty ] && [ -w /dev/tty ]; then
	exec </dev/tty
fi

# Locate repository root or clone on remote curl execution
REPO_ROOT=""
cur="${PWD:-.}"
while [ "$cur" != "/" ] && [ "$cur" != "." ]; do
	if [ -d "${cur}/manifests" ] && [ -f "${cur}/scripts/bootstrap_anx.sh" ]; then
		REPO_ROOT="$cur"
		break
	fi
	parent="$(dirname "$cur")"
	if [ "$parent" = "$cur" ]; then
		break
	fi
	cur="$parent"
done

if [ -z "${REPO_ROOT}" ]; then
	TARGET_REPO="${ANX_REPO_DIR:-${HOME}/.local/share/anx/repo}"
	echo "  [INFO] Remote curl execution detected: bootstrapping repository into ${TARGET_REPO}..."
	mkdir -p "$(dirname "${TARGET_REPO}")"
	if [ -d "${TARGET_REPO}/.git" ]; then
		git -C "${TARGET_REPO}" pull --rebase origin main 2>/dev/null || true
	else
		GIT_REMOTE_URL="${ANX_GIT_URL:-https://github.com/arturonavax/env.git}"
		if [ -d "${GIT_REMOTE_URL}" ]; then
			mkdir -p "${TARGET_REPO}"
			cp -r "${GIT_REMOTE_URL}/." "${TARGET_REPO}/"
		elif command -v git >/dev/null 2>&1; then
			git clone --depth 1 "${GIT_REMOTE_URL}" "${TARGET_REPO}"
		elif command -v curl >/dev/null 2>&1 && command -v tar >/dev/null 2>&1; then
			echo "  [INFO] 'git' not found; fetching repository archive via curl & tar..."
			mkdir -p "${TARGET_REPO}"
			curl -fsSL "https://github.com/arturonavax/env/archive/refs/heads/main.tar.gz" | tar -xz -C "${TARGET_REPO}" --strip-components=1
		elif command -v wget >/dev/null 2>&1 && command -v tar >/dev/null 2>&1; then
			echo "  [INFO] 'git' not found; fetching repository archive via wget & tar..."
			mkdir -p "${TARGET_REPO}"
			wget -qO- "https://github.com/arturonavax/env/archive/refs/heads/main.tar.gz" | tar -xz -C "${TARGET_REPO}" --strip-components=1
		else
			echo "[Installer Error]: 'git' or 'curl'+'tar' is required to fetch repository for installation." >&2
			exit 1
		fi
	fi
	REPO_ROOT="${TARGET_REPO}"
fi

cd "${REPO_ROOT}"

usage() {
	cat <<EOF
Usage: ./install.sh [OPTIONS] [COMMAND|PROFILE]

ANX Platform Installer & Environment Orchestrator (POSIX Entrypoint).

Modes & Commands:
  -w, --wizard, wizard          Launch interactive Bubble Tea v2 setup wizard (default in TTY).
  -y, --yes, --non-interactive  Run unattended non-interactive installation (zero prompts).
  -p, --plan, plan              Preview execution plan and touched files without modifying system.
  --check, check                Run platform diagnosis and health checks (doctor).
  --uninstall-plan              Preview uninstallation teardown plan.

Profiles & Targets:
  all                           Complete workstation environment (terminal, editor, devtools, ai).
  terminal                      Ghostty, Herdr multiplexer, lean Zsh, Starship, Atuin, and modern CLI tools.
  editor                        Neovim (LazyVim), LSP servers, Treesitter parsers.
  ai                            CLI AI assistants, RTK optimizer, MCP harnesses, and aitools suite.
  devtools                      Essential development toolchains and utilities.

Examples:
  sh ./install.sh               # Interactive guided setup (in TTY)
  sh ./install.sh --plan        # Preview installation plan (dry-run)
  sh ./install.sh all -y        # Unattended complete installation
  sh ./install.sh --check       # Run diagnostic health check
EOF
}

# Ensure bash is available for the compiler bootstrap engine
if ! command -v bash >/dev/null 2>&1; then
	echo "  [INFO] 'bash' is required by the pre-flight compiler engine; attempting automated setup..."
	if command -v apk >/dev/null 2>&1; then
		apk add --no-cache bash
	elif command -v pkg >/dev/null 2>&1; then
		pkg install -y bash
	elif command -v apt-get >/dev/null 2>&1; then
		apt-get update -y && apt-get install -y bash
	elif command -v dnf >/dev/null 2>&1; then
		dnf install -y bash
	elif command -v pacman >/dev/null 2>&1; then
		pacman -Sy --noconfirm bash
	else
		echo "[Installer Error]: 'bash' is required by bootstrap_anx.sh but could not be installed automatically." >&2
		exit 1
	fi
fi

# Translate CLI args for bootstrap_anx.sh / anx dispatch in pure POSIX sh
count=$#
while [ "$count" -gt 0 ]; do
	arg="$1"
	shift
	count=$((count - 1))
	case "$arg" in
	-p | --plan | plan | --dry-run | dry-run | --preview | preview)
		set -- "$@" "plan"
		;;
	--uninstall-plan | uninstall-plan)
		./uninstall.sh --plan
		exit 0
		;;
	-w | --wizard | wizard)
		set -- "$@" "wizard"
		;;
	--check | check)
		set -- "$@" "doctor"
		;;
	-h | --help | help)
		usage
		exit 0
		;;
	*)
		set -- "$@" "$arg"
		;;
	esac
done

if command -v bash >/dev/null 2>&1; then
	exec bash "./scripts/bootstrap_anx.sh" "$@"
else
	exec ./scripts/bootstrap_anx.sh "$@"
fi
