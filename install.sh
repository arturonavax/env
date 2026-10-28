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

source ./src/lib/strict.sh
source ./src/lib/colors.sh
source ./src/lib/logging.sh
source ./src/lib/platform.sh
source ./src/lib/packages.sh
source ./src/lib/backup.sh

usage() {
	cat <<EOF
Usage: ./install.sh [OPTIONS] [COMPONENTS...]

Modes:
  -w, --wizard        Launch interactive step-by-step setup wizard.

Components:
  requirements, r     Install development toolchains, languages (Go, Node, Rust, Python) and headers.
  fonts, f            Install patched mono Nerd Fonts.
  terminal, t         Install Ghostty, Herdr multiplexer, lean Zsh, Starship, Atuin, and modern CLI tools.
  editor, e           Install Neovim (LazyVim), language servers, and Treesitter parsers.
  ai                  Install AI CLI assistants (agy), RTK token killer, MCP servers, and harnesses.
  osconfig, o         Configure operating system desktop shortcuts, window switcher (rcmd), and tweaks.
  all, a              Install and configure all of the above subsystems.

Diagnostics & Help:
  --check             Run system diagnostic check and display platform summary without modifying files.
  -h, --help          Show this help message.

Examples:
  ./install.sh --wizard          # Interactive guided installation
  ./install.sh all               # Complete non-interactive installation
  ./install.sh terminal editor   # Install only terminal and editor suites
EOF
}

# Check for wizard invocation
for arg in "$@"; do
	case "$arg" in
	-w | --wizard | wizard)
		exec ./src/wizard.sh
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

# Parse Non-interactive Component Arguments
install_requirements=0
install_fonts=0
install_terminal=0
install_editor=0
install_ai=0
install_osconfig=0
install_all=0

for arg in "$@"; do
	case "$arg" in
	requirements | r) install_requirements=1 ;;
	fonts | f) install_fonts=1 ;;
	terminal | t) install_terminal=1 ;;
	editor | e) install_editor=1 ;;
	ai) install_ai=1 ;;
	osconfig | o) install_osconfig=1 ;;
	all | a) install_all=1 ;;
	*)
		log_error "Unrecognized argument: $arg"
		usage
		exit 1
		;;
	esac
done

if ((install_all == 1)); then
	install_requirements=1
	install_fonts=1
	install_terminal=1
	install_editor=1
	install_ai=1
	install_osconfig=1
fi

# Fallback: if no component selected, default to editor
if ((install_requirements == 0 && install_fonts == 0 && install_terminal == 0 && install_editor == 0 && install_ai == 0 && install_osconfig == 0)); then
	install_editor=1
fi

# Banner & Summary
log_box "arturonavax/env Installer" \
	"Platform: $(detect_os) ($(detect_distro) - $(detect_arch))" \
	"Sudo privileges will be acquired upfront." \
	"Automatic configuration snapshot will be generated."

# 1. Automatic Snapshot
log_step "Generating configuration snapshot in ~/.arturonavax-env-backups/..."
snapshot_dir="$(backup_create)"
log_info "Snapshot ready: $(basename "$snapshot_dir")"

# 2. Acquire Sudo Privileges & Start Keep-alive
log_step "Acquiring administrator credentials..."
ensure_sudo
log_success "Privileges confirmed."

# 3. Synchronize Repository if inside a git clone
if [[ -d ./.git ]]; then
	log_step "Synchronizing latest repository state from origin/main..."
	git pull origin main --rebase 2>/dev/null || log_warn "Git pull skipped (working directory dirty or offline)."
fi

# 4. Execute Selected Subsystems
if ((install_requirements == 1)); then
	log_step "Executing Requirements installer..."
	bash ./src/_install_requirements.sh
fi

if ((install_fonts == 1)); then
	log_step "Executing Fonts installer..."
	bash ./src/remotes/install_fonts.sh
fi

if ((install_terminal == 1)); then
	log_step "Executing Terminal installer..."
	bash ./src/install_terminal.sh
fi

if ((install_editor == 1)); then
	log_step "Executing Editor installer..."
	bash ./src/install_editor.sh
fi

if ((install_ai == 1)); then
	log_step "Executing AI ecosystem installer..."
	bash ./src/install_ai.sh
fi

if ((install_osconfig == 1)); then
	log_step "Applying operating system configurations..."
	if [[ "$(detect_os)" == "linux" ]]; then
		bash ./src/remotes/linux_osconfig.sh
	elif [[ "$(detect_os)" == "darwin" ]]; then
		bash ./src/remotes/macos_osconfig.sh
	fi
fi

# 5. macOS Homebrew Cleanup
if [[ "$(detect_os)" == "darwin" ]] && command -v brew >/dev/null 2>&1; then
	log_step "Running Homebrew cleanup..."
	brew cleanup 2>/dev/null || true
fi

log_box "Installation Finished Successfully!" \
	"Your development environment has been deployed." \
	"To activate changes, restart your terminal or run: source ~/.zshrc"
