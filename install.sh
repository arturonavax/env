#!/usr/bin/env bash
# ==============================================================================
# install.sh
# Primary multi-platform installation entry point for arturonavax/env.
# Bootstraps Go if needed and executes the ANX Platform orchestrator.
# Supports both local invocation and remote piped execution:
#   curl -fsSL https://env.arturonavax.dev/install.sh | bash
# ==============================================================================

set -euo pipefail

# If running through a pipe (e.g. curl ... | bash) and a controlling TTY exists,
# reconnect standard input so interactive wizards (Bubble Tea) can operate.
if [[ ! -t 0 ]] && [[ -r /dev/tty && -w /dev/tty ]]; then
	exec </dev/tty
fi

# Anchoring: locate repository root or clone on remote curl execution
REPO_ROOT=""
cur="$PWD"
while [[ "$cur" != "/" ]]; do
	if [[ -d "${cur}/manifests" && -f "${cur}/scripts/bootstrap_anx.sh" ]]; then
		REPO_ROOT="$cur"
		break
	fi
	cur="$(dirname "$cur")"
done

if [[ -z "${REPO_ROOT}" ]]; then
	TARGET_REPO="${ANX_REPO_DIR:-${HOME}/.local/share/anx/repo}"
	echo "  [INFO] Remote curl execution detected: bootstrapping repository into ${TARGET_REPO}..."
	mkdir -p "$(dirname "${TARGET_REPO}")"
	if [[ -d "${TARGET_REPO}/.git" ]]; then
		git -C "${TARGET_REPO}" pull --rebase origin main 2>/dev/null || true
	else
		GIT_REMOTE_URL="${ANX_GIT_URL:-https://github.com/arturonavax/env.git}"
		if [[ -d "${GIT_REMOTE_URL}" ]]; then
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
			echo "[Installer Error]: 'git' or 'curl'+'tar' is required to clone repository for installation." >&2
			exit 1
		fi
	fi
	REPO_ROOT="${TARGET_REPO}"
fi

cd "${REPO_ROOT}"

usage() {
	cat <<EOF
Usage: ./install.sh [OPTIONS] [COMMAND|PROFILE]

ANX Platform Installer & Environment Orchestrator.

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
  ./install.sh                  # Interactive guided setup (in TTY)
  ./install.sh --plan           # Preview installation plan (dry-run)
  ./install.sh all -y           # Unattended complete installation
  ./install.sh --check          # Run diagnostic health check
EOF
}

# Translate CLI args for bootstrap_anx.sh / anx dispatch
bootstrap_args=()
while [[ $# -gt 0 ]]; do
	case "$1" in
	-p | --plan | plan | --dry-run | dry-run | --preview | preview)
		bootstrap_args+=("plan")
		;;
	--uninstall-plan | uninstall-plan)
		./uninstall.sh --plan
		exit 0
		;;
	-w | --wizard | wizard)
		bootstrap_args+=("wizard")
		;;
	--check | check)
		bootstrap_args+=("doctor")
		;;
	-h | --help | help)
		usage
		exit 0
		;;
	*)
		bootstrap_args+=("$1")
		;;
	esac
	shift
done

exec ./scripts/bootstrap_anx.sh "${bootstrap_args[@]}"
