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

# Safe dispatch wrapper: re-connects standard input to controlling terminal
# only when handing off to child processes at the end of installation,
# preserving the incoming curl pipe during script execution.
dispatch_exec() {
	if [ -c /dev/tty ] && ( : </dev/tty ) 2>/dev/null; then
		exec "$@" </dev/tty
	else
		exec "$@"
	fi
}

DEST_BIN="${HOME}/.local/bin"
if [ -z "${ANX_INITIAL_PATH:-}" ]; then
	ANX_INITIAL_PATH="${PATH}"
	export ANX_INITIAL_PATH
fi

for _brew_bin in /opt/homebrew/bin /usr/local/bin; do
	if [ -d "${_brew_bin}" ]; then
		case ":${PATH}:" in
		*":${_brew_bin}:"*) ;;
		*) PATH="${_brew_bin}:${PATH}" ;;
		esac
	fi
done
export PATH

# Self-heal broken /etc/pam.d/sudo_local (e.g. missing /opt/homebrew/.../pam_reattach.so after an
# interrupted destroyer run) in user-space without sudo so that subsequent sudo calls can initialize PAM.
repair_macos_sudo_pam_if_needed() {
	[ "$(uname -s 2>/dev/null || true)" = "Darwin" ] || return 0
	[ -r /etc/pam.d/sudo_local ] || return 0
	_shim=""
	for _cand in /usr/lib/pam/pam_permit.so.2 /usr/lib/pam/pam_permit.so /usr/lib/pam/pam_deny.so.2 /usr/lib/pam/pam_deny.so; do
		if [ -f "$_cand" ]; then
			_shim="$_cand"
			break
		fi
	done
	[ -n "$_shim" ] || return 0
	while IFS= read -r _line || [ -n "$_line" ]; do
		case "$_line" in
		"" | \#*) continue ;;
		esac
		for _tok in $_line; do
			case "$_tok" in
			/*.so)
				if [ ! -f "$_tok" ]; then
					_dir="$(dirname "$_tok")"
					mkdir -p "$_dir" 2>/dev/null || true
					rm -f "$_tok" 2>/dev/null || true
					ln -sfn "$_shim" "$_tok" 2>/dev/null || true
				fi
				;;
			esac
		done
	done </etc/pam.d/sudo_local
}
repair_macos_sudo_pam_if_needed

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

# Terminal styling & Tokyo Night palette with graceful degradation
has_utf8=0
case "${LC_ALL:-${LC_CTYPE:-${LANG:-}}}" in
	*[Uu][Tt][Ff]-8* | *[Uu][Tt][Ff]8*) has_utf8=1 ;;
esac

TN_CYAN=""
TN_GREEN=""
TN_ORANGE=""
TN_RED=""
TN_ACCENT=""
TN_DIM=""
TN_BOLD=""
TN_RESET=""

# Detect controlling terminal for interactive execution
TTY_DEV=""
if [ -c /dev/tty ] && ( : </dev/tty ) 2>/dev/null; then
	TTY_DEV="/dev/tty"
fi

if [ -z "${NO_COLOR:-}" ] && [ "${TERM:-}" != "dumb" ]; then
	if [ "${COLORTERM:-}" = "truecolor" ] || [ "${COLORTERM:-}" = "24bit" ]; then
		# Tokyo Night 24-bit TrueColor (red accent #f7768e)
		TN_CYAN="\033[38;2;125;207;255m"
		TN_GREEN="\033[38;2;115;218;202m"
		TN_ORANGE="\033[38;2;255;158;100m"
		TN_RED="\033[38;2;247;118;142m"
		TN_ACCENT="$TN_RED"
		TN_DIM="\033[38;2;86;95;137m"
		TN_BOLD="\033[1m"
		TN_RESET="\033[0m"
	elif command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null || echo 0)" -ge 256 ] || case "${TERM:-}" in *256color*) true ;; *) false ;; esac; then
		# Tokyo Night 256-color palette
		TN_CYAN="\033[38;5;117m"
		TN_GREEN="\033[38;5;79m"
		TN_ORANGE="\033[38;5;215m"
		TN_RED="\033[38;5;204m"
		TN_ACCENT="$TN_RED"
		TN_DIM="\033[38;5;60m"
		TN_BOLD="\033[1m"
		TN_RESET="\033[0m"
	elif command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null || echo 0)" -ge 8 ] || [ -t 1 ] || [ -n "$TTY_DEV" ]; then
		# Standard 8/16-color ANSI fallback
		TN_CYAN="\033[1;36m"
		TN_GREEN="\033[1;32m"
		TN_ORANGE="\033[1;33m"
		TN_RED="\033[1;31m"
		TN_ACCENT="$TN_RED"
		TN_DIM="\033[2m"
		TN_BOLD="\033[1m"
		TN_RESET="\033[0m"
	fi
fi

if [ "$has_utf8" -eq 1 ]; then
	SYM_DOT="·"
	SYM_DIAMOND="◆"
else
	SYM_DOT="-"
	SYM_DIAMOND="*"
fi

prompt_out() {
	if [ -n "$TTY_DEV" ]; then
		printf "%b" "$*" >"$TTY_DEV"
	else
		printf "%b" "$*"
	fi
}

check_precompiled_release() {
	if [ "$FORCE_SOURCE" -eq 1 ] || [ "$FORCE_GO_INSTALL" -eq 1 ]; then
		return 1
	fi
	_arch="$(detect_os_arch 2>/dev/null)" || return 1
	_archive="anx_${_arch}.tar.gz"
	_url="${ANX_RELEASE_URL:-https://github.com/arturonavax/env/releases/latest/download/${_archive}}"
	if command -v curl >/dev/null 2>&1; then
		_status="$(curl -sIL -o /dev/null -w "%{http_code}" --connect-timeout 2 --max-time 3 "$_url" 2>/dev/null || echo "000")"
		[ "$_status" = "200" ] || [ "$_status" = "302" ]
	elif command -v wget >/dev/null 2>&1; then
		wget --spider -q -T 3 "$_url" 2>/dev/null
	else
		return 1
	fi
}

ANX_ONE_SHOT="${ANX_ONE_SHOT:-0}"
INTERACTIVE_TTY=0
if [ "$#" -eq 0 ] && [ -z "${ANX_INSTALL_MODE:-}" ]; then
	if [ -t 0 ] && [ -t 1 ]; then
		INTERACTIVE_TTY=1
	elif [ -n "$TTY_DEV" ]; then
		INTERACTIVE_TTY=1
	fi
fi

if [ "$INTERACTIVE_TTY" -eq 1 ]; then
	os_arch="$(detect_os_arch)" || exit 1

	has_release=0
	if check_precompiled_release; then
		has_release=1
	fi

	prompt_out "\n  ${TN_ACCENT}${SYM_DIAMOND}${TN_RESET}  ${TN_BOLD}ANX Setup${TN_RESET}  ${TN_DIM}${SYM_DOT}${TN_RESET}  ${TN_CYAN}${os_arch}${TN_RESET}\n"
	prompt_out "  ${TN_DIM}Choose installation method:${TN_RESET}\n"

	if [ "$has_release" -eq 1 ]; then
		prompt_out "    ${TN_ACCENT}${TN_BOLD}1)${TN_RESET} Verified release   ${TN_GREEN}(Fast · pre-compiled binary)${TN_RESET}\n"
		prompt_out "    ${TN_ACCENT}${TN_BOLD}2)${TN_RESET} Build from source  ${TN_ORANGE}(Compiles latest binary)${TN_RESET}\n"
		prompt_out "    ${TN_ACCENT}${TN_BOLD}3)${TN_RESET} Preview            ${TN_DIM}(Dry-run · no changes)${TN_RESET}\n"
		prompt_out "    ${TN_ACCENT}${TN_BOLD}4)${TN_RESET} Exit\n\n"
		prompt_out "  Select [1-4] (${TN_DIM}default: 1${TN_RESET}): "
		if [ -n "$TTY_DEV" ]; then
			read -r choice <"$TTY_DEV" || choice="1"
		else
			read -r choice || choice="1"
		fi
		prompt_out "\n"
		case "$choice" in
		2)
			FORCE_SOURCE=1
			ANX_ONE_SHOT=1
			set -- "wizard"
			;;
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
			FORCE_SOURCE=0
			ANX_ONE_SHOT=1
			set -- "wizard"
			;;
		esac
	else
		prompt_out "    ${TN_ACCENT}${TN_BOLD}1)${TN_RESET} Build from source  ${TN_ORANGE}(Compiles latest binary)${TN_RESET}\n"
		prompt_out "    ${TN_ACCENT}${TN_BOLD}2)${TN_RESET} Preview            ${TN_DIM}(Dry-run · no changes)${TN_RESET}\n"
		prompt_out "    ${TN_ACCENT}${TN_BOLD}3)${TN_RESET} Exit\n\n"
		prompt_out "  Select [1-3] (${TN_DIM}default: 1${TN_RESET}): "
		if [ -n "$TTY_DEV" ]; then
			read -r choice <"$TTY_DEV" || choice="1"
		else
			read -r choice || choice="1"
		fi
		prompt_out "\n"
		case "$choice" in
		2)
			echo "  [INFO] Preview mode selected."
			echo "  - OS/Arch     : ${os_arch}"
			echo "  - Destination : ${DEST_BIN}/anx"
			echo "  - Action      : Dry-run preview, zero modifications made."
			exit 0
			;;
		3)
			echo "  Installation aborted."
			exit 0
			;;
		*)
			# Default 1: build from source
			FORCE_SOURCE=1
			ANX_ONE_SHOT=1
			set -- "wizard"
			;;
		esac
	fi
fi

export ANX_ONE_SHOT
if [ "$FORCE_SOURCE" -eq 1 ]; then
	export ANX_FORCE_SOURCE=1
fi

has_usable_git() {
	if ! command -v git >/dev/null 2>&1; then
		return 1
	fi
	if [ "$(uname -s)" = "Darwin" ]; then
		_git_bin="$(command -v git 2>/dev/null || true)"
		if [ "$_git_bin" = "/usr/bin/git" ] || [ "$_git_bin" = "${ANX_SYSTEM_GIT_PATH:-/usr/bin/git}" ]; then
			_dev_dir="$(xcode-select -p 2>/dev/null || true)"
			if [ -z "$_dev_dir" ] || [ ! -x "${_dev_dir}/usr/bin/git" ]; then
				return 1
			fi
		fi
	fi
	return 0
}

fetch_repo_tarball() {
	_url="$1"
	_dest="$2"
	_tmp_tar="$(mktemp -d 2>/dev/null || mktemp -d -t anx-repo)"
	_archive="${_tmp_tar}/repo.tar.gz"
	_extract="${_tmp_tar}/src"
	mkdir -p "$_extract"

	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$_url" -o "$_archive" || {
			rm -rf "$_tmp_tar"
			return 1
		}
	elif command -v wget >/dev/null 2>&1; then
		wget -qO "$_archive" "$_url" || {
			rm -rf "$_tmp_tar"
			return 1
		}
	else
		rm -rf "$_tmp_tar"
		return 1
	fi

	if ! command -v tar >/dev/null 2>&1 || ! tar -xzf "$_archive" -C "$_extract" --strip-components=1 2>/dev/null; then
		rm -rf "$_tmp_tar"
		return 1
	fi
	if [ ! -f "${_extract}/scripts/bootstrap_anx.sh" ]; then
		rm -rf "$_tmp_tar"
		return 1
	fi

	mkdir -p "$_dest"
	cp -Rf "${_extract}/." "${_dest}/"
	rm -rf "$_tmp_tar"
	return 0
}

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

ensure_repo_checkout() {
	TARGET_REPO="${ANX_REPO_DIR:-${HOME}/.local/share/anx/repo}"
	echo "  [INFO] Bootstrapping repository into ${TARGET_REPO}..."
	mkdir -p "$(dirname "${TARGET_REPO}")"
	TARBALL_URL="${ANX_TARBALL_URL:-https://github.com/arturonavax/env/archive/refs/heads/main.tar.gz}"
	if [ -d "${TARGET_REPO}/.git" ] && has_usable_git; then
		git -C "${TARGET_REPO}" pull --rebase origin main 2>/dev/null || true
	elif [ -f "${TARGET_REPO}/scripts/bootstrap_anx.sh" ]; then
		fetch_repo_tarball "${TARBALL_URL}" "${TARGET_REPO}" || true
	else
		GIT_REMOTE_URL="${ANX_GIT_URL:-https://github.com/arturonavax/env.git}"
		if [ -d "${GIT_REMOTE_URL}" ]; then
			mkdir -p "${TARGET_REPO}"
			cp -r "${GIT_REMOTE_URL}/." "${TARGET_REPO}/"
		elif has_usable_git && git clone --depth 1 "${GIT_REMOTE_URL}" "${TARGET_REPO}"; then
			:
		elif fetch_repo_tarball "${TARBALL_URL}" "${TARGET_REPO}"; then
			:
		else
			echo "[Installer Error]: Could not fetch repository (requires usable 'git' or 'curl'/'wget' + 'tar')." >&2
			exit 1
		fi
	fi
	REPO_ROOT="${TARGET_REPO}"
}

# If remote execution (no local repo), ensure checkout; delegate Darwin and interactive one-shot runs through bootstrap_anx.sh
if [ -z "${REPO_ROOT}" ]; then
	if [ "$(uname -s)" != "Darwin" ] && [ "${ANX_ONE_SHOT:-0}" -eq 0 ]; then
		if [ "$FORCE_GO_INSTALL" -eq 1 ]; then
			if try_go_install_direct; then
				dispatch_exec "${DEST_BIN}/anx" "$@"
			fi
		fi

		if [ "$FORCE_SOURCE" -eq 0 ]; then
			if try_download_release_direct; then
				dispatch_exec "${DEST_BIN}/anx" "$@"
			fi
			if try_go_install_direct; then
				dispatch_exec "${DEST_BIN}/anx" "$@"
			fi
		fi
	fi

	# Fallback to source checkout/tarball (avoiding Apple /usr/bin/git stub on fresh macOS without CLT)
	ensure_repo_checkout
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
	dispatch_exec bash "./scripts/bootstrap_anx.sh" "$@"
else
	dispatch_exec ./scripts/bootstrap_anx.sh "$@"
fi
