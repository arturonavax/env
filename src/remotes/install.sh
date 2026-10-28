#!/usr/bin/env bash
# ==============================================================================
# src/remotes/install.sh
# Remote bootstrap entry point served from https://env.arturonavax.dev/install.sh
# Run: bash <(curl -fsSL "https://env.arturonavax.dev/install.sh") [options]
# ==============================================================================

set -euo pipefail

export REMOTE_PRIMARY_URL="${REMOTE_PRIMARY_URL:-https://env.arturonavax.dev}"
export REMOTE_FALLBACK_URL="${REMOTE_FALLBACK_URL:-https://environment.arturonavax.dev}"
installation_folder="${installation_folder:-$HOME/arturonavax-env}"

if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
	echo "[Installer Error]: 'curl' or 'wget' is required to execute remote bootstrap." >&2
	exit 1
fi

# Multi-Platform Git Bootstrapper
if ! command -v git >/dev/null 2>&1; then
	echo "[Remote Bootstrap]: Git is not installed. Bootstrapping Git..."

	case "$(uname -s)" in
	Linux)
		if [[ -f /etc/os-release ]]; then
			# shellcheck source=/dev/null
			. /etc/os-release
			id="${ID:-}"
			id_like="${ID_LIKE:-}"

			if [[ "$id_like" == *"debian"* || "$id_like" == *"ubuntu"* || "$id" == "debian" || "$id" == "ubuntu" ]]; then
				sudo apt update -y && sudo apt install -y git
			elif [[ "$id_like" == *"fedora"* || "$id_like" == *"rhel"* || "$id" == "fedora" || "$id" == "rhel" ]]; then
				sudo dnf install -y git 2>/dev/null || sudo yum install -y git
			elif [[ "$id_like" == *"arch"* || "$id" == "arch" || "$id" == "manjaro" || "$id" == "endeavouros" ]]; then
				sudo pacman -Sy --noconfirm git
			elif [[ "$id" == "alpine" ]]; then
				sudo apk add --no-cache git
			else
				echo "[Remote Bootstrap Error]: Unsupported Linux distribution: $id" >&2
				exit 1
			fi
		fi
		;;
	Darwin)
		# On macOS, git comes with Command Line Tools
		if [[ ! -e "/Library/Developer/CommandLineTools/usr/bin/git" ]]; then
			echo "[Remote Bootstrap]: Installing macOS Command Line Tools..."
			xcode-select --install 2>/dev/null || true
			until [[ -e "/Library/Developer/CommandLineTools/usr/bin/git" ]]; do
				sleep 5
			done
		fi
		;;
	*)
		echo "[Remote Bootstrap Error]: Unsupported operating system: $(uname -s)" >&2
		exit 1
		;;
	esac
fi

# Clone or Update Repository
if [[ -d "${installation_folder}/.git" ]]; then
	echo "[Remote Bootstrap]: Existing repository found in ${installation_folder}. Updating..."
	git -C "$installation_folder" pull --rebase origin main 2>/dev/null || true
else
	echo "[Remote Bootstrap]: Cloning repository into ${installation_folder}..."
	rm -rf "$installation_folder"
	git clone --depth 1 https://github.com/arturonavax/env.git "$installation_folder"
fi

cd "$installation_folder"

# Launch main installer
if [[ "$#" -eq 0 ]] && [[ -t 0 ]]; then
	exec ./install.sh --wizard
else
	exec ./install.sh "$@"
fi
