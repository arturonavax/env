#!/usr/bin/env bash
# ==============================================================================
# src/ai/install_aitools.sh
# Multi-platform installer for aitools AI Productivity & Quota Management Suite.
# Deploys aitools CLI, default configurations, and system integrations.
# Part of arturonavax/env development environment.
# ==============================================================================

# Anchoring
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "[aitools Error]: Repository root not found." >&2
		exit 1
	fi
	if ! cd ..; then
		echo "[aitools Error]: Failed to change directory." >&2
		exit 1
	fi
done

source ./src/lib/strict.sh
source ./src/lib/colors.sh
source ./src/lib/logging.sh
source ./src/lib/platform.sh
source ./src/lib/smart_append.sh
# shellcheck source=/dev/null
source ./src/lib/transaction.sh 2>/dev/null || true

log_step "Installing aitools AI Productivity & Quota Management Suite..."

DEST_USER_BIN="${DEST_BIN_DIR:-$HOME/.local/bin}"
CONFIG_DIR="$HOME/.config/aitools"
SRC_BIN="./files/ai/aitools/aitools"
SRC_CFG="./files/ai/aitools/config.default.json"

mkdir -p "$DEST_USER_BIN" "$CONFIG_DIR"

if [[ ! -f "$SRC_BIN" ]]; then
	log_error "Source binary not found: $SRC_BIN"
	exit 1
fi

# 1. Install executable binary into target user bin
cp -f "$SRC_BIN" "${DEST_USER_BIN}/aitools"
chmod +x "${DEST_USER_BIN}/aitools"

if command -v tx_record_file_created >/dev/null 2>&1; then
	tx_record_file_created "${DEST_USER_BIN}/aitools"
fi

# 2. Symlink to /usr/local/bin if root or passwordless sudo available
if [[ -w "/usr/local/bin" ]]; then
	ln -sf "${DEST_USER_BIN}/aitools" "/usr/local/bin/aitools" 2>/dev/null || true
elif command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
	sudo ln -sf "${DEST_USER_BIN}/aitools" "/usr/local/bin/aitools" 2>/dev/null || true
	if command -v tx_record_symlink_created >/dev/null 2>&1; then
		tx_record_symlink_created "/usr/local/bin/aitools"
	fi
fi

# 3. Deploy default configuration if none exists
if [[ ! -f "${CONFIG_DIR}/config.json" ]] && [[ -f "$SRC_CFG" ]]; then
	cp "$SRC_CFG" "${CONFIG_DIR}/config.json"
	log_info "Initialized default configuration at ${CONFIG_DIR}/config.json"
	if command -v tx_record_file_created >/dev/null 2>&1; then
		tx_record_file_created "${CONFIG_DIR}/config.json"
	fi
fi

# 4. Clean up legacy quota-primer systemd units if found
if command -v systemctl >/dev/null 2>&1; then
	systemctl --user stop quota-monitor.service quota-primer.timer quota-primer.service 2>/dev/null || true
	systemctl --user disable quota-monitor.service quota-primer.timer quota-primer.service 2>/dev/null || true
	rm -f "$HOME/.config/systemd/user/quota-monitor.service" \
		"$HOME/.config/systemd/user/quota-primer.service" \
		"$HOME/.config/systemd/user/quota-primer.timer" 2>/dev/null || true
	systemctl --user daemon-reload 2>/dev/null || true
fi

log_success "aitools installed successfully: ${DEST_USER_BIN}/aitools"
