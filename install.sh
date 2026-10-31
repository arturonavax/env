#!/bin/sh
# ==============================================================================
# install.sh
# Primary POSIX-compliant multi-platform installation entry point for arturonavax/env.
# Compatible with any POSIX shell (dash, ash, bash, zsh, ksh) on Linux, macOS,
# Raspberry Pi, Alpine, Android Termux, and BSD.
#
# Supports both local repository invocation and remote piped execution:
#   curl -fsSL https://sh.arturonavax.dev | sh
#   curl -fsSL https://env.arturonavax.dev | sh
# ==============================================================================

set -eu

# If running through a pipe (e.g. curl ... | sh) and a controlling TTY exists and can be opened,
# reconnect standard input so interactive wizards (Bubble Tea) can operate.
if [ ! -t 0 ] && [ -c /dev/tty ]; then
	if (exec </dev/tty) 2>/dev/null; then
		exec </dev/tty
	fi
fi

DEST_BIN="${HOME}/.local/bin"

# Locate repository root if running inside an existing checkout
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

# Detect whether source build, release download, or plan preview is explicitly requested
FORCE_SOURCE=0
FORCE_GO_INSTALL=0

case "${ANX_INSTALL_MODE:-}" in
source) FORCE_SOURCE=1 ;;
release) FORCE_SOURCE=0 ;;
plan)
	echo "  [INFO] Preview mode selected via ANX_INSTALL_MODE. No system changes made."
	exit 0
	;;
esac

for arg in "$@"; do
	case "$arg" in
	--from-source | --source | --mode=source)
		FORCE_SOURCE=1
		;;
	--mode=release)
		FORCE_SOURCE=0
		;;
	--go-install)
		FORCE_GO_INSTALL=1
		;;
	esac
done

detect_os_arch() {
	os="$(uname -s | tr '[:upper:]' '[:lower:]')"
	arch="$(uname -m)"
	case "$arch" in
	x86_64 | amd64) arch="amd64" ;;
	aarch64 | arm64) arch="arm64" ;;
	*)
		echo "  [ERROR] Unsupported architecture: ${arch}" >&2
		return 1
		;;
	esac
	echo "${os}_${arch}"
}

# Interactive TTY prompt (supports /dev/tty even when piped via curl | sh)
INTERACTIVE_TTY=0
TTY_DEV=""
if [ "$#" -eq 0 ] && [ -z "${ANX_INSTALL_MODE:-}" ]; then
	if [ -t 0 ] && [ -t 1 ]; then
		INTERACTIVE_TTY=1
	elif [ -c /dev/tty ] && [ -r /dev/tty ] && [ -w /dev/tty ]; then
		INTERACTIVE_TTY=1
		TTY_DEV="/dev/tty"
	fi
fi

if [ "$INTERACTIVE_TTY" -eq 1 ]; then
	os_arch="$(detect_os_arch)" || exit 1
	prompt_out() {
		if [ -n "$TTY_DEV" ]; then
			printf "%s" "$*" >"$TTY_DEV"
		else
			printf "%s" "$*"
		fi
	}
	prompt_out "\n  \033[1mANX Setup\033[0m  ·  ${os_arch}\n"
	prompt_out "  Choose installation method:\n"
	prompt_out "    1) Verified release  (Fast · no compiler needed)\n"
	prompt_out "    2) Build from source (Compiles latest binary)\n"
	prompt_out "    3) Preview           (Dry-run · no changes)\n"
	prompt_out "    4) Exit\n"
	prompt_out "  Select [1-4] (default: 1): "
	if [ -n "$TTY_DEV" ]; then
		read -r choice <"$TTY_DEV" || choice="1"
	else
		read -r choice || choice="1"
	fi
	case "$choice" in
	2) FORCE_SOURCE=1 ;;
	3)
		echo "  [INFO] Preview mode selected."
		echo "  - OS/Arch     : ${os_arch}"
		echo "  - Destination : ${DEST_BIN}/anx"
		echo "  - Action      : Dry-run preview, zero modifications made."
		exit 0
		;;
	4)
		echo "  Installation aborted."
		exit 0
		;;
	*)
		# Default 1: verified release
		;;
	esac
fi

try_download_release_direct() {
	if [ "$FORCE_SOURCE" -eq 1 ] || [ "$FORCE_GO_INSTALL" -eq 1 ]; then
		return 1
	fi

	os_arch="$(detect_os_arch)"
	archive_name="anx_${os_arch}.tar.gz"
	release_url="${ANX_RELEASE_URL:-https://github.com/arturonavax/env/releases/latest/download/${archive_name}}"
	default_sums_url="$(dirname "${release_url}")/SHA256SUMS"
	sums_url="${ANX_SUMS_URL:-${default_sums_url}}"

	if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
		return 1
	fi

	tmp_dir="$(mktemp -d 2>/dev/null || mktemp -d -t anx-release)"
	tarball="${tmp_dir}/${archive_name}"
	sums_file="${tmp_dir}/SHA256SUMS"

	dl_ok=0
	if command -v curl >/dev/null 2>&1; then
		if curl -fsSL --connect-timeout 4 --max-time 30 -o "$tarball" "$release_url" 2>/dev/null && \
		   curl -fsSL --connect-timeout 4 --max-time 15 -o "$sums_file" "$sums_url" 2>/dev/null; then
			dl_ok=1
		fi
	elif command -v wget >/dev/null 2>&1; then
		if wget -q -T 30 -O "$tarball" "$release_url" 2>/dev/null && \
		   wget -q -T 15 -O "$sums_file" "$sums_url" 2>/dev/null; then
			dl_ok=1
		fi
	fi

	if [ "$dl_ok" -eq 1 ] && [ -s "$tarball" ] && [ -s "$sums_file" ]; then
		# Checksum verification (fail-closed)
		want_sum="$(grep "${archive_name}" "$sums_file" 2>/dev/null | awk '{print $1}' || true)"
		if [ -z "$want_sum" ]; then
			echo "  [WARN] Release asset checksum not found in SHA256SUMS." >&2
			rm -rf "$tmp_dir"
			return 1
		fi
		if ! echo "$want_sum" | grep -Eq '^[0-9a-fA-F]{64}$'; then
			echo "  [WARN] Invalid checksum format for release asset." >&2
			rm -rf "$tmp_dir"
			return 1
		fi

		computed_sum=""
		if command -v sha256sum >/dev/null 2>&1; then
			computed_sum="$(sha256sum "$tarball" | awk '{print $1}')"
		elif command -v shasum >/dev/null 2>&1; then
			computed_sum="$(shasum -a 256 "$tarball" | awk '{print $1}')"
		else
			echo "  [WARN] Neither sha256sum nor shasum available to verify checksum." >&2
			rm -rf "$tmp_dir"
			return 1
		fi

		if [ "$computed_sum" != "$want_sum" ]; then
			echo "  [WARN] Release asset checksum mismatch. Computed: $computed_sum, expected: $want_sum." >&2
			rm -rf "$tmp_dir"
			return 1
		fi

		# Verify tarball paths are safe (no leading / or ../)
		if tar -tf "$tarball" 2>/dev/null | grep -E '(^/|\.\./)' >/dev/null; then
			echo "  [WARN] Hostile path detected in release archive. Aborting release extraction." >&2
			rm -rf "$tmp_dir"
			return 1
		fi

		mkdir -p "${tmp_dir}/extract"
		if tar -xzf "$tarball" -C "${tmp_dir}/extract" 2>/dev/null; then
			if [ -f "${tmp_dir}/extract/anx" ]; then
				chmod 0755 "${tmp_dir}/extract/anx"
				if "${tmp_dir}/extract/anx" version >/dev/null 2>&1; then
					mkdir -p "$DEST_BIN"
					mv -f "${tmp_dir}/extract/anx" "${DEST_BIN}/anx"
					ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/aitools"
					ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/safe-suspend"
					rm -rf "$tmp_dir"
					echo "  [OK] ANX verified release binary deployed to ${DEST_BIN}/anx."
					return 0
				fi
			fi
		fi
	fi

	rm -rf "$tmp_dir"
	return 1
}

try_go_install_direct() {
	if [ "$FORCE_SOURCE" -eq 1 ] && [ "$FORCE_GO_INSTALL" -eq 0 ]; then
		return 1
	fi

	if command -v go >/dev/null 2>&1; then
		echo "  [INFO] Installing ANX via 'go install github.com/arturonavax/env/cmd/anx@latest'..."
		if go install -trimpath github.com/arturonavax/env/cmd/anx@latest; then
			gobin="$(go env GOBIN)"
			[ -z "$gobin" ] && gobin="$(go env GOPATH)/bin"
			if [ -x "${gobin}/anx" ]; then
				mkdir -p "$DEST_BIN"
				if [ "${gobin}" != "${DEST_BIN}" ]; then
					ln -sf "${gobin}/anx" "${DEST_BIN}/anx"
				fi
				ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/aitools"
				ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/safe-suspend"
				echo "  [OK] ANX successfully installed via 'go install'."
				return 0
			fi
		fi
	fi
	return 1
}

# If remote execution (no local repo), prioritize pre-compiled release, then go-install, then source clone
if [ -z "${REPO_ROOT}" ]; then
	if [ "$FORCE_GO_INSTALL" -eq 1 ]; then
		if try_go_install_direct; then
			exec "${DEST_BIN}/anx" "$@"
		fi
	fi

	if [ "$FORCE_SOURCE" -eq 0 ]; then
		if try_download_release_direct; then
			exec "${DEST_BIN}/anx" "$@"
		fi
		if try_go_install_direct; then
			exec "${DEST_BIN}/anx" "$@"
		fi
	fi

	# Fallback to source clone
	TARGET_REPO="${ANX_REPO_DIR:-${HOME}/.local/share/anx/repo}"
	echo "  [INFO] Bootstrapping repository into ${TARGET_REPO}..."
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
			mkdir -p "${TARGET_REPO}"
			curl -fsSL "https://github.com/arturonavax/env/archive/refs/heads/main.tar.gz" | tar -xz -C "${TARGET_REPO}" --strip-components=1
		elif command -v wget >/dev/null 2>&1 && command -v tar >/dev/null 2>&1; then
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
  --from-source                 Force compilation from local source repository.
  --go-install                  Install ANX using standard 'go install'.
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
  sh ./install.sh --go-install  # Install via 'go install'
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
