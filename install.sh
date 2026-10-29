#!/usr/bin/env bash
# ==============================================================================
# install.sh
# Primary multi-platform installation entry point for arturonavax/env.
# Supports both unattended automated provisioning and interactive wizard mode.
# Run: ./install.sh --help
# ==============================================================================

# Anchoring
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "[Installer Error]: Repository root not found." >&2
		exit 1
	fi
	if ! cd ..; then
		echo "[Installer Error]: Failed to change directory." >&2
		exit 1
	fi
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

# Early check for plan, wizard, check, help, and unattended flags
for arg in "$@"; do
	case "$arg" in
	-p | --plan | plan | --dry-run | dry-run | --preview | preview)
		# Filter out plan flags and pass target components to show_install_plan
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
	-y | --yes | -n | --non-interactive | --unattended | --batch)
		exec ./src/wizard.sh "$@"
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
