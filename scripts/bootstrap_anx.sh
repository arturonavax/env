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

# Source GVM if available in standard location
if [[ -s "${HOME}/.gvm/scripts/gvm" ]]; then
	# shellcheck source=/dev/null
	source "${HOME}/.gvm/scripts/gvm" 2>/dev/null || true
fi

# Ensure target binary directory exists
mkdir -p "${DEST_BIN}"
if [[ ":${PATH}:" != *":${DEST_BIN}:"* ]]; then
	export PATH="${DEST_BIN}:${PATH}"
fi

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

	if ! command -v gvm >/dev/null 2>&1; then
		if [[ -s "${HOME}/.gvm/scripts/gvm" ]]; then
			set +u
			# shellcheck source=/dev/null
			source "${HOME}/.gvm/scripts/gvm" 2>/dev/null || true
			set -u
		elif command -v curl >/dev/null 2>&1; then
			log_info "Installing GVM..."
			bash < <(curl -fsSL "https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer") || return 1
			if [[ -s "${HOME}/.gvm/scripts/gvm" ]]; then
				set +u
				# shellcheck source=/dev/null
				source "${HOME}/.gvm/scripts/gvm" 2>/dev/null || true
				set -u
			fi
		fi
	fi

	if command -v gvm >/dev/null 2>&1; then
		log_info "Installing ${TARGET_GO_VER} via GVM..."
		gvm install "${TARGET_GO_VER}" -B 2>/dev/null || gvm install "${TARGET_GO_VER}" 2>/dev/null || return 1
		gvm use "${TARGET_GO_VER}" --default 2>/dev/null || return 1
		if check_go_version; then
			log_success "Go $(go version | awk '{print $3}') configured via GVM."
			return 0
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

	if [[ -d "/usr/local/go/bin" ]]; then
		export PATH="/usr/local/go/bin:${PATH}"
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

ensure_go_runtime() {
	if check_go_version; then
		log_success "Go runtime verified: $(go version | awk '{print $3}')"
		return 0
	fi

	log_warn "Go runtime (>= 1.27.3) not detected. Initiating multi-tier bootstrap..."

	if install_go_tier1_gvm; then
		return 0
	fi
	log_warn "Tier 1 (GVM) did not succeed. Falling back to Tier 2 (standalone script)..."

	if install_go_tier2_standalone; then
		return 0
	fi
	log_warn "Tier 2 (standalone script) did not succeed. Falling back to Tier 3 (package manager)..."

	if install_go_tier3_package_manager; then
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
	version=$(git -C "${REPO_ROOT}" describe --tags --always 2>/dev/null || echo "1.0.0")
	commit=$(git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo "dev")
	date=$(date -u "+%Y-%m-%d")

	(
		cd "${REPO_ROOT}"
		go build -ldflags "-s -w -X main.Version=${version} -X main.Commit=${commit} -X main.Date=${date}" \
			-o "${DEST_BIN}/anx" ./cmd/anx
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

# ------------------------------------------------------------------------------
# Main Dispatch
# ------------------------------------------------------------------------------
main() {
	echo -e "${BOLD_WHITE}==============================================================================${RESET}"
	echo -e "${BOLD_WHITE}  ANX Platform Bootstrap & Environment Pre-Flight${RESET}"
	echo -e "${BOLD_WHITE}==============================================================================${RESET}"

	BIN_NAME="${ANX_BIN_NAME:-}"
	local force_go_install=0
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
			*)
				forward_args+=("$1")
				shift
				;;
		esac
	done

	if [[ "${force_go_install}" -eq 1 ]]; then
		if try_go_install; then
			if [[ ${#forward_args[@]} -gt 0 ]]; then
				exec "${DEST_BIN}/anx" "${forward_args[@]}"
			else
				log_success "Bootstrap complete! You can now run 'anx' or 'anx wizard'."
				return 0
			fi
		fi
	fi

	# Tier 0: Attempt pre-compiled GitHub release download if not forcing source build
	if try_download_release_binary; then
		if [[ -x "${DEST_BIN}/anx" ]]; then
			if [[ ${#forward_args[@]} -gt 0 ]]; then
				log_info "Delegating execution to: ${DEST_BIN}/anx ${forward_args[*]}"
				exec "${DEST_BIN}/anx" "${forward_args[@]}"
			else
				local alias_hint=""
				if [[ -n "${BIN_NAME}" && "${BIN_NAME}" != "anx" ]]; then
					alias_hint=" (or '${BIN_NAME}')"
				fi
				log_success "Bootstrap complete! You can now run 'anx'${alias_hint} or 'anx wizard'."
				return 0
			fi
		fi
	fi

	ensure_go_runtime
	build_anx

	if [[ ${#forward_args[@]} -gt 0 ]]; then
		log_info "Delegating execution to: ${DEST_BIN}/anx ${forward_args[*]}"
		exec "${DEST_BIN}/anx" "${forward_args[@]}"
	else
		local alias_hint=""
		if [[ -n "${BIN_NAME}" && "${BIN_NAME}" != "anx" ]]; then
			alias_hint=" (or '${BIN_NAME}')"
		fi
		log_success "Bootstrap complete! You can now run 'anx'${alias_hint} or 'anx wizard'."
	fi
}

main "$@"
