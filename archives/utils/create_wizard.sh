#!/usr/bin/env bash
# ==============================================================================
# utils/create_wizard.sh
# Production Pure-Bash Wizard Generator for arturonavax/env.
# Generates zero-dependency interactive wizard scripts using wizard_engine.sh.
# Usage:
#   ./utils/create_wizard.sh <target_script_path> ["Wizard Title"]
# ==============================================================================

set -euo pipefail

# Anchoring
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "Error: Repository root not found." >&2
		exit 1
	fi
	if ! cd ..; then
		echo "Error: Failed to change directory." >&2
		exit 1
	fi
done

target_file="${1:-}"
wizard_title="${2:-New Setup Wizard}"

if [[ -z "$target_file" ]]; then
	echo "Usage: $0 <target_script_path.sh> [\"Wizard Title\"]" >&2
	echo "Example: $0 ./scripts/provision_db.sh \"Database Provisioning Wizard\"" >&2
	exit 1
fi

mkdir -p "$(dirname "$target_file")"

cat <<'EOF' >"$target_file"
#!/usr/bin/env bash
# ==============================================================================
# TARGET_TITLE
# Generated via arturonavax/env wizard_engine.
# ==============================================================================

# Anchoring loop to repository root
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "Error: Repository root not found." >&2
		exit 1
	fi
	if ! cd ..; then
		echo "Error: Failed to change directory." >&2
		exit 1
	fi
done

# Source strict mode and pure-bash wizard engine
# shellcheck source=/dev/null
source "./src/lib/strict.sh"
# shellcheck source=/dev/null
source "./src/lib/wizard_engine.sh"

run_wizard() {
	# 1. Initialize Wizard (Title, Total Stages, Subtitle)
	wizard_init "TARGET_TITLE" 3 "Interactive setup guide"
	wizard_banner

	# ── Stage 1 ───────────────────────────────────────────────────────────────
	wizard_stage "Initial Requirements & Context"
	say "Verifying base credentials and workspace environment..."
	step "Ensure you have repository write access and valid tokens."
	note "Tip: Values will be persisted safely without committing secrets to git."
	pause

	# ── Stage 2 ───────────────────────────────────────────────────────────────
	wizard_stage "Configuration & Credentials"
	say "Please configure your environment parameters:"
	ask APP_ENV "Select target environment" "development"
	ask_secret API_KEY "Enter API authentication key"

	# Persist configuration
	wizard_write_env "APP_ENV" "$APP_ENV"
	step "Configuration validated and recorded."
	pause

	# ── Stage 3 ───────────────────────────────────────────────────────────────
	wizard_stage "Deployment & Verification"
	say "Executing final service verification..."
	if confirm "Proceed with deployment?" "y"; then
		step "Deploying services..."
		# Add your automated actions here
	else
		warn "Deployment skipped by user choice."
	fi

	# ── Final Summary Frame ───────────────────────────────────────────────────
	wizard_finish "Setup completed successfully!"
}

run_wizard "$@"
EOF

# Substitute title
sed -i "s/TARGET_TITLE/${wizard_title}/g" "$target_file"
chmod +x "$target_file"

echo "Generated interactive wizard in: $target_file"
echo "Syntax validation: $(bash -n "$target_file" && echo 'PASSED' || echo 'FAILED')"
