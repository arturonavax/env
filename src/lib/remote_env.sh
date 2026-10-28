#!/usr/bin/env bash
# ==============================================================================
# src/lib/remote_env.sh
# Dual Local/Remote Resolver for env.arturonavax.dev subdomain and CDN scripts.
# Part of arturonavax/env development environment.
# ==============================================================================

# Prevent double sourcing
[[ -n "${_REMOTE_ENV_SH_LOADED:-}" ]] && return 0
_REMOTE_ENV_SH_LOADED=1

export REMOTE_PRIMARY_URL="${REMOTE_PRIMARY_URL:-https://env.arturonavax.dev}"
export REMOTE_FALLBACK_URL="${REMOTE_FALLBACK_URL:-https://environment.arturonavax.dev}"
export repo_remote_files="$REMOTE_PRIMARY_URL"

# Discover if running from inside local cloned repo
_is_local_repo() {
	[[ -d "./files" && -d "./src" ]] || [[ -d "../files" && -d "../src" ]]
}

# Resolve full local path or remote URL for a script
resolve_asset() {
	local relative_path="$1"

	# 1. Check current directory
	if [[ -f "./${relative_path}" ]]; then
		echo "./${relative_path}"
		return 0
	fi

	# 2. Check parent directory if nested
	if [[ -f "../${relative_path}" ]]; then
		echo "../${relative_path}"
		return 0
	fi

	# 3. Check inside src/
	if [[ -f "./src/${relative_path}" ]]; then
		echo "./src/${relative_path}"
		return 0
	fi

	# 4. Fallback to Primary CDN
	echo "${REMOTE_PRIMARY_URL}/${relative_path}"
}

# Source a library file either locally or remotely
source_env_lib() {
	local lib_name="$1"
	local local_candidate="./src/lib/${lib_name}.sh"

	if [[ -f "$local_candidate" ]]; then
		# shellcheck source=/dev/null
		source "$local_candidate"
		return 0
	fi

	# Remote source with fallback
	local remote_url="${REMOTE_PRIMARY_URL}/lib/${lib_name}.sh"
	if curl --proto '=https' --tlsv1.2 -fsSL "$remote_url" 2>/dev/null | bash -s --; then
		return 0
	fi

	local fallback_url="${REMOTE_FALLBACK_URL}/lib/${lib_name}.sh"
	curl --proto '=https' --tlsv1.2 -fsSL "$fallback_url" | bash -s --
}
