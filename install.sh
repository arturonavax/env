#!/usr/bin/env bash
# ==============================================================================
# install.sh
# Primary multi-platform installation entry point for arturonavax/env.
# Supports both unattended automated provisioning and interactive wizard mode.
# Run: ./install.sh --help
# ==============================================================================

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
		else
			echo "[Installer Error]: 'git' is required to clone repository for installation." >&2
			exit 1
		fi
	fi
	REPO_ROOT="${TARGET_REPO}"
fi

cd "${REPO_ROOT}"

source ./src/lib/strict.sh
source ./src/lib/colors.sh
source ./src/lib/logging.sh
source ./src/lib/platform.sh
source ./src/lib/packages.sh
source ./src/lib/backup.sh
source ./src/lib/plan.sh

usage() {
	cat <<EOF
Usage: ./install.sh [OPTIONS] [COMPONENTS...]

Modes:
  -w, --wizard                  Launch interactive step-by-step setup wizard (default in TTY).
  -y, --yes, --non-interactive  Run unattended non-interactive installation (zero prompts).

Components:
  requirements, r               Install development toolchains, languages (Go, Node, Rust, Python) and headers.
  fonts, f                      Install patched mono Nerd Fonts.
  terminal, t                   Install Ghostty, Herdr multiplexer, lean Zsh, Starship, Atuin, and modern CLI tools.
  editor, e                     Install Neovim (LazyVim), language servers, and Treesitter parsers.
  ai                            Install AI CLI assistants (agy), RTK token killer, MCP servers, and harnesses.
  osconfig, o                   Configure operating system desktop shortcuts, window switcher (rcmd), and tweaks.
  all, a                        Install and configure all of the above subsystems.

Configuration & Paths:
  --dest-bin=<path>             Target directory for user binaries (default: ~/.local/bin).
  --backup-dir=<path>           Target directory for backup snapshots (default: ~/.arturonavax-env-backups).
  --skip-backup                 Bypass initial configuration snapshot creation.
  --go-manager=<strategy>       Golang manager strategy: gvm (default), standalone, package.
  --node-manager=<manager>      Node.js package manager: pnpm (default), npm.
  --rollback-on-error           Automatically trigger transactional rollback on any stage failure.

Planning & Dry Run:
  -p, --plan, plan              Display complete installation plan, translated packages, and touched files.
  --dry-run, preview            Alias for --plan (safe preview without modifying the filesystem).
  --uninstall-plan              Display teardown plan of packages, binaries, and caches purged on uninstall.

Diagnostics & Help:
  --check                       Run system diagnostic check and display platform summary without modifying files.
  -h, --help                    Show this help message.

Examples:
  ./install.sh --plan            # Preview full installation manifest and touched files
  ./install.sh plan terminal     # Preview installation plan for terminal subsystem only
  ./install.sh --uninstall-plan  # Preview uninstallation plan and purged files
  ./install.sh --wizard          # Interactive guided installation
  ./install.sh all               # Complete non-interactive installation
  ./install.sh terminal editor   # Install only terminal and editor suites
EOF
}

# Early check for plan, wizard, check, and help flags
for arg in "$@"; do
	case "$arg" in
	-p | --plan | plan | --dry-run | dry-run | --preview | preview)
		plan_components=()
		for item in "$@"; do
			case "$item" in
			-p | --plan | plan | --dry-run | dry-run | --preview | preview) ;;
			*) plan_components+=("$item") ;;
			esac
		done
		show_install_plan "${plan_components[@]}"
		exit 0
		;;
	--uninstall-plan | uninstall-plan)
		show_uninstall_plan
		exit 0
		;;
	-w | --wizard | wizard)
		shift_args=()
		for item in "$@"; do
			case "$item" in
			-w | --wizard | wizard) ;;
			*) shift_args+=("$item") ;;
			esac
		done
		exec ./src/wizard.sh "${shift_args[@]}"
		;;
	--check | check)
		print_platform_summary
		exit 0
		;;
	-h | --help | help)
		usage
		exit 0
		;;
	esac
done

# Check if legacy bash mode was explicitly requested
LEGACY_BASH=0
filtered_args=()
for arg in "$@"; do
	if [[ "$arg" == "--legacy-bash" ]]; then
		LEGACY_BASH=1
	else
		filtered_args+=("$arg")
	fi
done

if [[ "${LEGACY_BASH}" -ne 1 ]]; then
	exec ./scripts/bootstrap_anx.sh "${filtered_args[@]}"
fi

# If invoked interactively with no arguments, launch wizard by default!
if [[ "$#" -eq 0 ]] && [[ -t 0 ]]; then
	exec ./src/wizard.sh
fi

# Parse component targets and flags, delegating to transactional wizard runner
selected_components=()
other_flags=()

for arg in "$@"; do
	case "$arg" in
	requirements | r) selected_components+=("requirements") ;;
	fonts | f) selected_components+=("fonts") ;;
	terminal | t) selected_components+=("terminal") ;;
	editor | e) selected_components+=("editor") ;;
	ai) selected_components+=("ai") ;;
	osconfig | o) selected_components+=("osconfig") ;;
	all | a) selected_components+=("all") ;;
	*) other_flags+=("$arg") ;;
	esac
done

if [[ ${#selected_components[@]} -gt 0 ]]; then
	comp_joined="$(IFS=','; echo "${selected_components[*]}")"
	exec ./src/wizard.sh -y --components="$comp_joined" "${other_flags[@]}"
fi

# Fallback: if no component selected, default to editor
exec ./src/wizard.sh -y --components=editor "${other_flags[@]}"
