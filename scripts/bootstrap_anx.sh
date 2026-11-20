#!/usr/bin/env bash
# ==============================================================================
# scripts/bootstrap_anx.sh
# Multi-platform Pre-Flight Analyzer & Golang Bootstrap for ANX Platform.
#
# Strategy:
#   0. Tier 0: Pre-compiled verified release download (instant, zero compiler needed)
#   1. Tier 0.5: Direct 'go install' if Go is present and requested
#   2. Pre-flight check for Go (>= 1.27.3)
#   3. If missing or outdated:
#      - Tier 1: GVM (Go Version Manager) binary release
#      - Tier 2: Personal standalone installer (src/remotes/install_golang.sh -i)
#      - Tier 3: Native package manager (brew, pacman, dnf, apt, apk)
#   4. Compile ANX binary with embedded version metadata
#   5. Symlink 'safe-suspend' -> 'anx' for multicall binary compatibility
#   6. Delegate forwarded arguments to ANX CLI
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
readonly REPO_ROOT

MIN_GO_MAJOR=1
MIN_GO_MINOR=27
MIN_GO_PATCH=3
TARGET_GO_VER="go1.27.3"
DEST_BIN="${HOME}/.local/bin"

# ANSI Colors
BOLD_WHITE="\033[1;37m"
GREEN="\033[0;32m"
YELLOW="\033[0;33m"
CYAN="\033[0;36m"
RED="\033[0;31m"
RESET="\033[0m"

log_info()    { echo -e "  [${CYAN}INFO${RESET}] $*"; }
log_step()    { echo -e "  [${BOLD_WHITE}STEP${RESET}] $*"; }
log_success() { echo -e "  [${GREEN}OK${RESET}] $*"; }
log_warn()    { echo -e "  [${YELLOW}WARN${RESET}] $*"; }
log_error()   { echo -e "  [${RED}ERROR${RESET}] $*" >&2; }

if [[ -z "${ANX_INITIAL_PATH:-}" ]]; then
	ANX_INITIAL_PATH="${PATH}"
	export ANX_INITIAL_PATH
fi

# Upstream GVM references unset variables (e.g. GVM_DEBUG, GVM_NO_GIT_BAK, ZSH_VERSION)
# and overrides builtin 'cd' with an unguarded shell function. Relax strict flags while
# loading or invoking GVM and restore builtin 'cd' immediately afterwards.
source_gvm_safely() {
	[[ -s "${HOME}/.gvm/scripts/gvm" ]] || return 1
	local rc=0
	export GVM_ROOT="${GVM_ROOT:-${HOME}/.gvm}"
	export GVM_DEBUG="${GVM_DEBUG:-0}"
	export GVM_NO_GIT_BAK="${GVM_NO_GIT_BAK:-1}"
	set +euo pipefail
	# shellcheck source=/dev/null
	source "${HOME}/.gvm/scripts/gvm" >/dev/null 2>&1 || rc=$?
	unset -f cd 2>/dev/null || true
	set -euo pipefail
	return "${rc}"
}

run_gvm_safely() {
	command -v gvm >/dev/null 2>&1 || return 1
	local rc=0
	export GVM_ROOT="${GVM_ROOT:-${HOME}/.gvm}"
	export GVM_DEBUG="${GVM_DEBUG:-0}"
	export GVM_NO_GIT_BAK="${GVM_NO_GIT_BAK:-1}"
	set +euo pipefail
	gvm "$@" >/dev/null 2>&1 || rc=$?
	unset -f cd 2>/dev/null || true
	set -euo pipefail
	return "${rc}"
}

run_gvm_isolated() {
	command -v gvm >/dev/null 2>&1 || return 1
	local rc=0
	(
		set +euo pipefail
		export GVM_ROOT="${GVM_ROOT:-${HOME}/.gvm}"
		export GVM_DEBUG="${GVM_DEBUG:-0}"
		export GVM_NO_GIT_BAK="${GVM_NO_GIT_BAK:-1}"
		gvm "$@" >/dev/null 2>&1
	) || rc=$?
	return "${rc}"
}

# Source GVM if available in standard location without breaking strict mode
if [[ -s "${HOME}/.gvm/scripts/gvm" ]]; then
	source_gvm_safely || true
fi

ensure_macos_homebrew_env() {
	if command -v brew >/dev/null 2>&1; then
		return 0
	fi
	if [[ -x "/opt/homebrew/bin/brew" ]]; then
		eval "$(/opt/homebrew/bin/brew shellenv)"
	elif [[ -x "/usr/local/bin/brew" ]]; then
		eval "$(/usr/local/bin/brew shellenv)"
	fi
}

if [[ "$(uname -s)" == "Darwin" ]]; then
	ensure_macos_homebrew_env
fi
GO_SYS_BIN="${ANX_GO_SYS_BIN:-/usr/local/go/bin}"
if ! command -v go >/dev/null 2>&1 && [[ -d "${GO_SYS_BIN}" && ":${PATH}:" != *":${GO_SYS_BIN}:"* ]]; then
	export PATH="${GO_SYS_BIN}:${PATH}"
fi

# Ensure target binary directory exists
mkdir -p "${DEST_BIN}"
if [[ ":${PATH}:" != *":${DEST_BIN}:"* ]]; then
	export PATH="${DEST_BIN}:${PATH}"
fi

CLT_PLACEHOLDER="${ANX_CLT_PLACEHOLDER:-/tmp/.com.apple.dt.CommandLineTools.installondemand.in-progress}"
CLT_PLACEHOLDER_CREATED=0

cleanup_bootstrap() {
	if [[ "${CLT_PLACEHOLDER_CREATED:-0}" -eq 1 ]]; then
		rm -f "${CLT_PLACEHOLDER}" >/dev/null 2>&1 || sudo -n rm -f "${CLT_PLACEHOLDER}" >/dev/null 2>&1 || true
	fi
}
trap cleanup_bootstrap EXIT INT TERM

has_macos_clt() {
	local dev_dir
	dev_dir="$(xcode-select -p 2>/dev/null || true)"
	if [[ -n "${dev_dir}" && -x "${dev_dir}/usr/bin/git" ]]; then
		return 0
	fi
	[[ -x "/Library/Developer/CommandLineTools/usr/bin/git" ]]
}

has_usable_git() {
	if ! command -v git >/dev/null 2>&1; then
		return 1
	fi
	if [[ "$(uname -s)" == "Darwin" ]]; then
		local git_bin
		git_bin="$(command -v git 2>/dev/null || true)"
		if [[ "${git_bin}" == "/usr/bin/git" || "${git_bin}" == "${ANX_SYSTEM_GIT_PATH:-/usr/bin/git}" ]] && ! has_macos_clt; then
			return 1
		fi
	fi
	return 0
}

pick_latest_clt_label() {
	awk '
		/^[[:space:]]*\* Label:[[:space:]]*.*Command Line Tools/ {
			label = $0
			sub(/^[[:space:]]*\* Label:[[:space:]]*/, "", label)
			ver = label
			sub(/^.*-/, "", ver)
			n = split(ver, p, ".")
			key = sprintf("%05d.%05d.%05d", (n >= 1 ? p[1] + 0 : 0), (n >= 2 ? p[2] + 0 : 0), (n >= 3 ? p[3] + 0 : 0))
			if (best_key == "" || key >= best_key) {
				best_key = key
				best_label = label
			}
		}
		END {
			if (best_label != "") {
				print best_label
			}
		}
	'
}

repair_macos_sudo_pam_if_needed() {
	if [[ "$(uname -s)" != "Darwin" ]] || [[ ! -r /etc/pam.d/sudo_local ]]; then
		return 0
	fi
	local shim="" cand line tok dir
	for cand in /usr/lib/pam/pam_permit.so.2 /usr/lib/pam/pam_permit.so /usr/lib/pam/pam_deny.so.2 /usr/lib/pam/pam_deny.so; do
		if [[ -f "${cand}" ]]; then
			shim="${cand}"
			break
		fi
	done
	[[ -n "${shim}" ]] || return 0

	while IFS= read -r line || [[ -n "${line}" ]]; do
		case "${line}" in
		"" | \#*) continue ;;
		esac
		for tok in ${line}; do
			case "${tok}" in
			/*.so)
				if [[ ! -f "${tok}" ]]; then
					dir="$(dirname "${tok}")"
					mkdir -p "${dir}" 2>/dev/null || true
					rm -f "${tok}" 2>/dev/null || true
					ln -sfn "${shim}" "${tok}" 2>/dev/null || true
				fi
				;;
			esac
		done
	done </etc/pam.d/sudo_local
}

ensure_macos_clt() {
	if [[ "$(uname -s)" != "Darwin" ]] || [[ "${ANX_SKIP_CLT:-0}" -eq 1 ]]; then
		return 0
	fi
	repair_macos_sudo_pam_if_needed
	if has_macos_clt; then
		return 0
	fi

	log_step "Xcode Command Line Tools not found; installing headlessly (no GUI dialog required)..."
	if ! sudo -n true >/dev/null 2>&1; then
		if [[ -c /dev/tty ]] && ( : </dev/tty ) 2>/dev/null; then
			if ! { sudo -v; } </dev/tty; then
				log_warn "Administrator privileges declined; skipping headless CLT setup and falling back to CGO_ENABLED=0 Go build."
				return 0
			fi
		elif [[ -t 0 ]]; then
			if ! sudo -v; then
				log_warn "Administrator privileges unavailable; continuing with CGO_ENABLED=0 Go build."
				return 0
			fi
		else
			log_warn "Non-interactive session without passwordless sudo; skipping headless CLT setup and continuing with CGO_ENABLED=0."
			return 0
		fi
	fi

	if ! sudo touch "${CLT_PLACEHOLDER}"; then
		log_warn "Could not create softwareupdate sentinel; continuing with CGO_ENABLED=0 fallback."
		return 0
	fi
	CLT_PLACEHOLDER_CREATED=1

	local label_clt
	label_clt="$(softwareupdate -l 2>/dev/null | pick_latest_clt_label || true)"
	if [[ -z "${label_clt}" ]]; then
		log_warn "Command Line Tools label not found via softwareupdate (run 'xcode-select --install' or retry online later). Continuing with CGO_ENABLED=0."
		cleanup_bootstrap
		CLT_PLACEHOLDER_CREATED=0
		return 0
	fi

	log_info "Installing macOS package: ${label_clt}..."
	if ! sudo softwareupdate -i "${label_clt}" </dev/null; then
		log_warn "softwareupdate install did not succeed for '${label_clt}'. Continuing with CGO_ENABLED=0."
	fi

	sudo rm -f "${CLT_PLACEHOLDER}" >/dev/null 2>&1 || rm -f "${CLT_PLACEHOLDER}" >/dev/null 2>&1 || true
	CLT_PLACEHOLDER_CREATED=0

	if [[ -d "/Library/Developer/CommandLineTools" ]]; then
		sudo xcode-select --switch /Library/Developer/CommandLineTools >/dev/null 2>&1 || true
	fi

	if has_macos_clt; then
		log_success "Xcode Command Line Tools verified."
	else
		log_warn "Command Line Tools still unavailable; continuing with pure-Go CGO_ENABLED=0 build."
	fi
}

ensure_macos_homebrew() {
	if [[ "$(uname -s)" != "Darwin" ]] || [[ "${ANX_SKIP_HOMEBREW:-0}" -eq 1 ]]; then
		return 0
	fi
	ensure_macos_homebrew_env
	if command -v brew >/dev/null 2>&1; then
		return 0
	fi
	if ! has_macos_clt || ! command -v curl >/dev/null 2>&1; then
		log_warn "Skipping automatic Homebrew setup (requires Xcode CLT and curl)."
		return 0
	fi

	log_step "Homebrew not detected on macOS; installing Homebrew..."
	local tmp_brew
	tmp_brew="$(mktemp "${TMPDIR:-/tmp}/anx-brew-install.XXXXXX")"
	if curl -fsSL "https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh" -o "${tmp_brew}"; then
		if [[ -c /dev/tty ]] && ( : </dev/tty ) 2>/dev/null; then
			{ NONINTERACTIVE=1 /bin/bash "${tmp_brew}"; } </dev/tty || log_warn "Homebrew installation failed; native macOS package installs will wait until 'brew' is installed."
		else
			NONINTERACTIVE=1 /bin/bash "${tmp_brew}" || log_warn "Homebrew installation failed; native macOS package installs will wait until 'brew' is installed."
		fi
	else
		log_warn "Could not download Homebrew installer script."
	fi
	rm -f "${tmp_brew}"
	ensure_macos_homebrew_env
}

# ------------------------------------------------------------------------------
# Pre-Compiled GitHub Release Binary
# ------------------------------------------------------------------------------
detect_os_arch() {
	local os arch
	os="$(uname -s | tr '[:upper:]' '[:lower:]')"
	arch="$(uname -m)"
	case "${arch}" in
		x86_64|amd64) arch="amd64" ;;
		aarch64|arm64) arch="arm64" ;;
		*) arch="amd64" ;;
	esac
	echo "${os}_${arch}"
}

try_download_release_binary() {
	if [[ "${ANX_FORCE_SOURCE:-0}" -eq 1 ]]; then
		log_info "ANX_FORCE_SOURCE=1: Skipping release download, building from source."
		return 1
	fi

	local os_arch archive_name release_url default_sums_url sums_url
	os_arch="$(detect_os_arch)"
	archive_name="anx_${os_arch}.tar.gz"
	release_url="${ANX_RELEASE_URL:-https://github.com/arturonavax/env/releases/latest/download/${archive_name}}"
	default_sums_url="$(dirname "${release_url}")/SHA256SUMS"
	sums_url="${ANX_SUMS_URL:-${default_sums_url}}"

	log_step "Checking for pre-compiled verified ANX release binary (${os_arch})..."
	if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
		return 1
	fi

	local tmp_dir
	tmp_dir="$(mktemp -d 2>/dev/null || mktemp -d -t anx-release)"
	local tarball="${tmp_dir}/${archive_name}"
	local sums_file="${tmp_dir}/SHA256SUMS"

	local dl_ok=0
	if command -v curl >/dev/null 2>&1; then
		if curl -fsSL --connect-timeout 4 --max-time 30 -o "${tarball}" "${release_url}" 2>/dev/null && \
		   curl -fsSL --connect-timeout 4 --max-time 15 -o "${sums_file}" "${sums_url}" 2>/dev/null; then
			dl_ok=1
		fi
	elif command -v wget >/dev/null 2>&1; then
		if wget -q -T 30 -O "${tarball}" "${release_url}" 2>/dev/null && \
		   wget -q -T 15 -O "${sums_file}" "${sums_url}" 2>/dev/null; then
			dl_ok=1
		fi
	fi

	if [[ "${dl_ok}" -eq 1 && -s "${tarball}" && -s "${sums_file}" ]]; then
		# Checksum verification before extraction
		local want_sum computed_sum
		want_sum=$(grep "${archive_name}" "${sums_file}" 2>/dev/null | awk '{print $1}' || true)
		if [[ -n "${want_sum}" ]]; then
			if command -v sha256sum >/dev/null 2>&1; then
				computed_sum=$(sha256sum "${tarball}" | awk '{print $1}')
			elif command -v shasum >/dev/null 2>&1; then
				computed_sum=$(shasum -a 256 "${tarball}" | awk '{print $1}')
			fi

			if [[ -n "${computed_sum:-}" && "${computed_sum}" != "${want_sum}" ]]; then
				log_warn "Checksum verification failed for ${archive_name}. Expected ${want_sum}, got ${computed_sum}"
				rm -rf "${tmp_dir}"
				return 1
			fi
			log_success "Checksum verified: ${want_sum}"
		fi

		# Safe isolated extraction
		local extract_dir="${tmp_dir}/extracted"
		mkdir -p "${extract_dir}"
		if tar -xzf "${tarball}" -C "${extract_dir}" 2>/dev/null; then
			if [[ -f "${extract_dir}/anx" ]]; then
				chmod 0755 "${extract_dir}/anx"
				# Self-test before placing into DEST_BIN
				if "${extract_dir}/anx" version >/dev/null 2>&1; then
					mv -f "${extract_dir}/anx" "${DEST_BIN}/anx"
					ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/aitools"
					ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/safe-suspend"
					if [[ -n "${BIN_NAME:-}" && "${BIN_NAME}" != "anx" ]]; then
						ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/${BIN_NAME}"
					fi
					rm -rf "${tmp_dir}"
					log_success "ANX verified release binary successfully deployed to ${DEST_BIN}/anx"
					return 0
				fi
			fi
		fi
	fi

	rm -rf "${tmp_dir}"
	log_info "Pre-compiled release not available or unverified. Proceeding with Go build."
	return 1
}

try_go_install() {
	if command -v go >/dev/null 2>&1; then
		log_step "Installing ANX via 'go install github.com/arturonavax/env/cmd/anx@latest'..."
		if go install -trimpath github.com/arturonavax/env/cmd/anx@latest; then
			local gobin
			gobin="$(go env GOBIN)"
			[[ -z "${gobin}" ]] && gobin="$(go env GOPATH)/bin"
			if [[ -x "${gobin}/anx" ]]; then
				if [[ "${gobin}" != "${DEST_BIN}" ]]; then
					ln -sf "${gobin}/anx" "${DEST_BIN}/anx"
				fi
				ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/aitools"
				ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/safe-suspend"
				if [[ -n "${BIN_NAME:-}" && "${BIN_NAME}" != "anx" ]]; then
					ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/${BIN_NAME}"
				fi
				log_success "ANX successfully installed via 'go install' and linked to ${DEST_BIN}/anx"
				return 0
			fi
		fi
	fi
	return 1
}

# ------------------------------------------------------------------------------
# Version Validation
# ------------------------------------------------------------------------------
check_go_version() {
	if ! command -v go >/dev/null 2>&1; then
		return 1
	fi
	local raw_ver
	raw_ver=$(go version 2>/dev/null | awk '{print $3}' | sed 's/^go//')
	local major minor patch
	major=$(echo "${raw_ver}" | cut -d. -f1)
	minor=$(echo "${raw_ver}" | cut -d. -f2)
	patch=$(echo "${raw_ver}" | cut -d. -f3 | sed 's/[^0-9].*$//')
	patch="${patch:-0}"

	if [[ -z "${major}" || -z "${minor}" ]]; then
		return 1
	fi

	if [[ "${major}" -gt "${MIN_GO_MAJOR}" ]]; then
		return 0
	fi
	if [[ "${major}" -eq "${MIN_GO_MAJOR}" && "${minor}" -gt "${MIN_GO_MINOR}" ]]; then
		return 0
	fi
	if [[ "${major}" -eq "${MIN_GO_MAJOR}" && "${minor}" -eq "${MIN_GO_MINOR}" && "${patch}" -ge "${MIN_GO_PATCH}" ]]; then
		return 0
	fi
	return 1
}

# ------------------------------------------------------------------------------
# Go Installation Tiers
# ------------------------------------------------------------------------------
install_go_tier1_gvm() {
	log_step "Tier 1: Attempting Go installation via GVM (Go Version Manager)..."

	if ! has_usable_git; then
		log_info "Usable git not detected; skipping Tier 1 (GVM) in favor of standalone Go binary installer."
		return 1
	fi

	if ! command -v gvm >/dev/null 2>&1; then
		if [[ -s "${HOME}/.gvm/scripts/gvm" ]]; then
			source_gvm_safely || true
		fi
		if ! command -v gvm >/dev/null 2>&1 && command -v curl >/dev/null 2>&1; then
			if [[ -d "${HOME}/.gvm" && ! -s "${HOME}/.gvm/scripts/gvm" ]]; then
				rm -rf "${HOME}/.gvm"
			fi
			log_info "Installing GVM..."
			local tmp_gvm_installer tmp_gvm_log gvm_installer_url
			tmp_gvm_installer="$(mktemp "${TMPDIR:-/tmp}/anx-gvm-installer.XXXXXX")"
			tmp_gvm_log="$(mktemp "${TMPDIR:-/tmp}/anx-gvm-log.XXXXXX")"
			gvm_installer_url="${ANX_GVM_INSTALLER_URL:-https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer}"
			if curl -fsSL "${gvm_installer_url}" -o "${tmp_gvm_installer}"; then
				GVM_NO_UPDATE_PROFILE=1 bash "${tmp_gvm_installer}" >"${tmp_gvm_log}" 2>&1 || true
			fi
			rm -f "${tmp_gvm_installer}" "${tmp_gvm_log}"
			source_gvm_safely || true
		fi
	fi

	if command -v gvm >/dev/null 2>&1; then
		log_info "Installing ${TARGET_GO_VER} via GVM..."
		if run_gvm_isolated install "${TARGET_GO_VER}" -B && run_gvm_safely use "${TARGET_GO_VER}" --default; then
			if check_go_version; then
				log_success "Go $(go version | awk '{print $3}') configured via GVM."
				return 0
			fi
		fi
	fi
	return 1
}

install_go_tier2_standalone() {
	log_step "Tier 2: Attempting Go installation via personal standalone installer..."
	local script_path="${REPO_ROOT}/scripts/install_golang.sh"
	if [[ -f "${script_path}" ]]; then
		bash "${script_path}" -i || return 1
	elif command -v curl >/dev/null 2>&1; then
		curl -fsSL "https://env.arturonavax.dev/install_golang.sh" | bash -s -- -i || return 1
	else
		return 1
	fi

	if [[ -d "${GO_SYS_BIN}" ]]; then
		export PATH="${GO_SYS_BIN}:${PATH}"
	fi

	if check_go_version; then
		log_success "Go $(go version | awk '{print $3}') installed via standalone script."
		return 0
	fi
	return 1
}

install_go_tier3_package_manager() {
	log_step "Tier 3: Attempting Go installation via native package manager..."
	if command -v brew >/dev/null 2>&1; then
		brew install go || return 1
	elif command -v pacman >/dev/null 2>&1; then
		sudo pacman -S --noconfirm go || return 1
	elif command -v dnf >/dev/null 2>&1; then
		sudo dnf install -y golang || return 1
	elif command -v apt-get >/dev/null 2>&1; then
		sudo apt-get update -qq && sudo apt-get install -y golang-go || return 1
	elif command -v apk >/dev/null 2>&1; then
		sudo apk add --no-cache go || return 1
	else
		return 1
	fi

	if check_go_version; then
		log_success "Go $(go version | awk '{print $3}') installed via package manager."
		return 0
	fi
	return 1
}

expose_go_binaries_in_local_bin() {
	local go_bin="" go_dir tool
	if command -v go >/dev/null 2>&1; then
		go_bin="$(command -v go)"
	elif [[ -s "${HOME}/.gvm/scripts/gvm" ]]; then
		source_gvm_safely || true
		go_bin="$(command -v go 2>/dev/null || true)"
	fi
	if [[ -z "${go_bin}" || ! -x "${go_bin}" ]] && [[ -x "${GO_SYS_BIN}/go" ]]; then
		go_bin="${GO_SYS_BIN}/go"
	fi
	[[ -n "${go_bin}" && -x "${go_bin}" ]] || return 0

	go_dir="$(dirname "${go_bin}")"
	mkdir -p "${HOME}/.local/bin"
	for tool in go gofmt; do
		if [[ -x "${go_dir}/${tool}" ]] && [[ "${go_dir}/${tool}" != "${HOME}/.local/bin/${tool}" ]]; then
			ln -sfn "${go_dir}/${tool}" "${HOME}/.local/bin/${tool}"
		fi
	done
}

ensure_go_runtime() {
	if check_go_version; then
		expose_go_binaries_in_local_bin
		log_success "Go runtime verified: $(go version | awk '{print $3}')"
		return 0
	fi

	log_warn "Go runtime (>= 1.27.3) not detected. Initiating multi-tier bootstrap..."

	if install_go_tier1_gvm; then
		expose_go_binaries_in_local_bin
		return 0
	fi
	log_warn "Tier 1 (GVM) did not succeed. Falling back to Tier 2 (standalone script)..."

	if install_go_tier2_standalone; then
		expose_go_binaries_in_local_bin
		return 0
	fi
	log_warn "Tier 2 (standalone script) did not succeed. Falling back to Tier 3 (package manager)..."

	if install_go_tier3_package_manager; then
		expose_go_binaries_in_local_bin
		return 0
	fi

	log_error "Could not bootstrap Go (>= 1.27.3) on this platform. Please install Go manually from https://go.dev/dl/."
	exit 1
}

# ------------------------------------------------------------------------------
# Build and Link ANX
# ------------------------------------------------------------------------------
build_anx() {
	log_step "Compiling ANX platform binary..."
	local version commit date
	version="1.0.0"
	commit="dev"
	if [[ -d "${REPO_ROOT}/.git" ]] && has_usable_git; then
		version=$(git -C "${REPO_ROOT}" describe --tags --always 2>/dev/null || echo "1.0.0")
		commit=$(git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo "dev")
	fi
	date=$(date -u "+%Y-%m-%d")

	(
		cd "${REPO_ROOT}"
		if [[ "$(uname -s)" == "Darwin" ]] && ! has_macos_clt; then
			CGO_ENABLED=0 go build -ldflags "-s -w -X main.Version=${version} -X main.Commit=${commit} -X main.Date=${date}" \
				-o "${DEST_BIN}/anx" ./cmd/anx
		else
			go build -ldflags "-s -w -X main.Version=${version} -X main.Commit=${commit} -X main.Date=${date}" \
				-o "${DEST_BIN}/anx" ./cmd/anx
		fi
	)

	# Multicall compatibility symlinks
	ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/aitools"
	ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/safe-suspend"
	log_success "ANX binary built and deployed to ${DEST_BIN}/anx (multicall symlinks: aitools, safe-suspend)"

	if [[ -n "${BIN_NAME:-}" && "${BIN_NAME}" != "anx" ]]; then
		ln -sf "${DEST_BIN}/anx" "${DEST_BIN}/${BIN_NAME}"
		log_success "Created custom binary alias: ${DEST_BIN}/${BIN_NAME} -> anx"
	fi
}

shell_rc_signature() {
	local f sig=""
	for f in "${HOME}/.zshrc" "${HOME}/.bashrc" "${HOME}/.base.zsh" "${HOME}/.base.bash" "${HOME}/.tools.sh" "${HOME}/.zshenv"; do
		if [[ -f "${f}" ]]; then
			sig="${sig}${f}:$(cksum < "${f}" 2>/dev/null || echo present);"
		fi
	done
	printf '%s' "${sig}"
}

launch_updated_shell_if_needed() {
	if [[ "${ANX_ONE_SHOT:-0}" -ne 1 || "${ANX_NO_SHELL_RELOAD:-0}" -eq 1 ]]; then
		return 0
	fi
	if ! [[ -c /dev/tty ]] || ! ( : </dev/tty ) 2>/dev/null; then
		return 0
	fi

	local target_shell=""
	if [[ "$(uname -s)" == "Darwin" ]]; then
		if [[ -x "/bin/zsh" ]]; then
			target_shell="/bin/zsh"
		elif [[ -n "${SHELL:-}" && -x "${SHELL}" ]]; then
			target_shell="${SHELL}"
		fi
	else
		if [[ -n "${SHELL:-}" && -x "${SHELL}" ]]; then
			target_shell="${SHELL}"
		elif command -v zsh >/dev/null 2>&1 && [[ -s "${HOME}/.zshrc" ]]; then
			target_shell="$(command -v zsh)"
		elif command -v bash >/dev/null 2>&1; then
			target_shell="$(command -v bash)"
		fi
	fi

	if [[ -z "${target_shell}" ]]; then
		return 0
	fi

	log_success "Configuration applied. Activating updated shell (no terminal restart needed)..."
	case "${target_shell##*/}" in
		zsh)
			exec "${target_shell}" -l </dev/tty
			;;
		*)
			exec "${target_shell}" -i </dev/tty
			;;
	esac
}

execute_anx_target() {
	if [[ $# -eq 0 ]]; then
		local alias_hint=""
		if [[ -n "${BIN_NAME:-}" && "${BIN_NAME}" != "anx" ]]; then
			alias_hint=" (or '${BIN_NAME}')"
		fi
		log_success "Bootstrap complete! You can now run 'anx'${alias_hint} or 'anx wizard'."
		return 0
	fi

	log_info "Delegating execution to: ${DEST_BIN}/anx $*"
	if [[ "${ANX_ONE_SHOT:-0}" -eq 1 && "${ANX_NO_SHELL_RELOAD:-0}" -eq 0 && -c /dev/tty ]] && ( : </dev/tty ) 2>/dev/null; then
		local rc_before rc_after cmd_rc=0
		rc_before="$(shell_rc_signature)"
		"${DEST_BIN}/anx" "$@" </dev/tty || cmd_rc=$?
		if [[ "${cmd_rc}" -ne 0 ]]; then
			return "${cmd_rc}"
		fi
		rc_after="$(shell_rc_signature)"
		if [[ "${rc_after}" != "${rc_before}" ]]; then
			launch_updated_shell_if_needed
		elif [[ ":${ANX_INITIAL_PATH:-}:" != *":${DEST_BIN}:"* && -s "${HOME}/.zshrc" ]]; then
			log_info "Run 'source ~/.zshrc' to load updated PATH in this existing shell session."
		fi
		return 0
	fi

	exec "${DEST_BIN}/anx" "$@"
}

# ------------------------------------------------------------------------------
# Main Dispatch
# ------------------------------------------------------------------------------
main() {
	echo -e "${BOLD_WHITE}==============================================================================${RESET}"
	echo -e "${BOLD_WHITE}  ANX Platform Bootstrap & Environment Pre-Flight${RESET}"
	echo -e "${BOLD_WHITE}==============================================================================${RESET}"

	BIN_NAME="${ANX_BIN_NAME:-}"
	local force_go_install=0
	local ensure_go_only=0
	local forward_args=()

	while [[ $# -gt 0 ]]; do
		case "$1" in
			--bin-name|-n|--name)
				BIN_NAME="$2"
				shift 2
				;;
			--bin-name=*)
				BIN_NAME="${1#*=}"
				shift
				;;
			--from-source|--force-source)
				ANX_FORCE_SOURCE=1
				shift
				;;
			--go-install)
				force_go_install=1
				shift
				;;
			--ensure-go-only)
				ensure_go_only=1
				shift
				;;
			*)
				forward_args+=("$1")
				shift
				;;
		esac
	done

	local readonly_action=0
	if [[ ${#forward_args[@]} -gt 0 ]]; then
		local arg
		for arg in "${forward_args[@]}"; do
			case "${arg}" in
				plan|--plan|-p|doctor|--check|--dry-run)
					readonly_action=1
					;;
			esac
		done
	fi

	if [[ ${#forward_args[@]} -eq 0 && "${ensure_go_only}" -eq 0 && -z "${ANX_BOOTSTRAP_ONLY:-}" ]]; then
		if [[ "${ANX_ONE_SHOT:-0}" -eq 1 ]] || [[ -t 0 && -t 1 ]]; then
			ANX_ONE_SHOT=1
			export ANX_ONE_SHOT
			forward_args=("wizard")
		fi
	fi

	if [[ "$(uname -s)" == "Darwin" && "${readonly_action}" -eq 0 ]]; then
		repair_macos_sudo_pam_if_needed
		ensure_macos_clt
		ensure_macos_homebrew
	fi

	if [[ "${ensure_go_only}" -eq 1 ]]; then
		ensure_go_runtime
		return 0
	fi

	if [[ "${force_go_install}" -eq 1 ]]; then
		if try_go_install; then
			if [[ ${#forward_args[@]} -gt 0 ]]; then
				execute_anx_target "${forward_args[@]}"
			else
				execute_anx_target
			fi
			return 0
		fi
	fi

	# Tier 0: Attempt pre-compiled GitHub release download if not forcing source build
	if try_download_release_binary; then
		if [[ -x "${DEST_BIN}/anx" ]]; then
			if [[ ${#forward_args[@]} -gt 0 ]]; then
				execute_anx_target "${forward_args[@]}"
			else
				execute_anx_target
			fi
			return 0
		fi
	fi

	ensure_go_runtime
	build_anx

	if [[ ${#forward_args[@]} -gt 0 ]]; then
		execute_anx_target "${forward_args[@]}"
	else
		execute_anx_target
	fi
}

main "$@"
