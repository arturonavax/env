#!/usr/bin/env bash
# ==============================================================================
# utils/aitools_wizard.sh
# Production Pure-Bash Interactive Setup Wizard for aitools AI Suite.
# Powered by wizard_engine.sh with zero external runtime dependencies.
# Part of arturonavax/env development environment.
# ==============================================================================

# Anchoring loop to repository root
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "[Error]: Repository root not found." >&2
		exit 1
	fi
	if ! cd ..; then
		echo "[Error]: Failed to change directory to parent." >&2
		exit 1
	fi
done

# shellcheck source=/dev/null
source "./src/lib/strict.sh"
# shellcheck source=/dev/null
source "./src/lib/wizard_engine.sh"

run_aitools_wizard() {
	wizard_init "aitools AI Suite · Setup Wizard" 5 "Interactive configuration for harnesses, models, templates & RTC daemon"
	wizard_banner

	local selected_backend="agy"
	local selected_model=""
	local selected_effort="low"
	local selected_template="conventional"
	local selected_stage_all="false"
	local selected_auto_wake="false"
	local selected_wake_time="04:00"
	local selected_resuspend="true"

	# ── Stage 1: AI Harness & Backend Selection ────────────────────────────────
	wizard_stage "AI Harness & CLI Backend Selection"
	say "Choose the default AI CLI backend used for commit generation and pipelines."
	say "Installed status detected in your current PATH:"

	local -a backend_names=("agy" "claude" "codex" "llm" "ollama" "opencode" "pi")
	local -a backend_menu=()

	local b
	for b in "${backend_names[@]}"; do
		if command -v "$b" >/dev/null 2>&1; then
			backend_menu+=("${b} (installed)")
		else
			backend_menu+=("${b} (not in PATH)")
		fi
	done

	local chosen_backend_entry=""
	select_option chosen_backend_entry "Select preferred AI harness:" "${backend_menu[@]}"

	# Extract clean name
	selected_backend="${chosen_backend_entry%% *}"
	step "Selected backend: ${BOLD_WHITE}${selected_backend}${RESET}"
	pause

	# ── Stage 2: Model & Reasoning Effort ─────────────────────────────────────
	wizard_stage "Model Configuration & Reasoning Effort"
	say "Configure default model name and reasoning effort for ${BOLD_WHITE}${selected_backend}${RESET}."

	case "$selected_backend" in
	agy)
		note "Popular agy models: gemini-3.8-flash-low, gemini-2.5-flash, claude-sonnet-5-5-low"
		;;
	claude)
		note "Popular Claude models: claude-3-7-sonnet, claude-3-5-sonnet, claude-3-5-haiku"
		;;
	codex)
		note "Popular Codex models: o3-mini, gpt-4o, gpt-4o-mini"
		;;
	llm)
		note "Popular llm models: gpt-4o, claude-3-5-sonnet, gemini-2.5-flash"
		;;
	ollama)
		note "Popular Ollama models: codellama, llama3, qwen2.5-coder, deepseek-coder"
		;;
	*)
		note "Leave blank to use the backend's default model."
		;;
	esac

	ask selected_model "Enter model identifier (or press Enter for default)" ""

	local -a effort_options=("low" "medium" "high" "xhigh")
	select_option selected_effort "Select reasoning effort level:" "${effort_options[@]}"
	step "Model: ${BOLD_WHITE}${selected_model:-default}${RESET} | Effort: ${BOLD_WHITE}${selected_effort}${RESET}"
	pause

	# ── Stage 3: Conventional Commit Template & Staging ───────────────────────
	wizard_stage "Git Commit Template & Staging Behavior"
	say "Select the default Conventional Commit prompt template format."

	local -a template_options=(
		"conventional - Standard (<type>(<scope>): <summary>)"
		"jira         - Enterprise Jira prefix ([<JIRA-ID>] <type>(<scope>): <summary>)"
		"jira-scope   - Jira ticket as scope (<type>(<JIRA-ID>): <summary>)"
		"detailed     - Multi-line with summary, WHAT/WHY paragraph and bullets"
	)
	local chosen_tmpl_entry=""
	select_option chosen_tmpl_entry "Select commit message template:" "${template_options[@]}"

	selected_template="$(echo "$chosen_tmpl_entry" | awk '{print $1}')"
	step "Selected template: ${BOLD_WHITE}${selected_template}${RESET}"

	if confirm "Stage all tracked modified files by default on commit (git add -u)?" "n"; then
		selected_stage_all="true"
	else
		selected_stage_all="false"
	fi
	pause

	# ── Stage 4: Hardware RTC Auto-Wake & Suspension ──────────────────────────
	wizard_stage "Hardware RTC Auto-Wake & Quota Priming"
	say "Supervise background quota priming and motherboard hardware RTC alarm."

	if command -v rtcwake >/dev/null 2>&1; then
		step "Hardware rtcwake binary detected in system."
	else
		warn "rtcwake not detected; auto-wake requires Linux RTC hardware support."
	fi

	if confirm "Enable scheduled RTC hardware auto-wake and quota priming?" "n"; then
		selected_auto_wake="true"
		ask selected_wake_time "Target wake time in 24h format (HH:MM)" "04:00"
		if confirm "Automatically put system back to sleep (suspend) after priming?" "y"; then
			selected_resuspend="true"
		else
			selected_resuspend="false"
		fi
	else
		selected_auto_wake="false"
	fi
	step "Auto-Wake: ${BOLD_WHITE}${selected_auto_wake}${RESET} (Time: ${selected_wake_time}, Resuspend: ${selected_resuspend})"
	pause

	# ── Stage 5: Save & Verification ──────────────────────────────────────────
	wizard_stage "Save Configuration"
	say "Writing preferences to ~/.config/aitools/config.json..."

	python3 -c "
import json, os, shutil

config_path = os.path.expanduser('~/.config/aitools/config.json')
os.makedirs(os.path.dirname(config_path), exist_ok=True)

data = {}
if os.path.exists(config_path):
    try:
        from importlib.machinery import SourceFileLoader
        aitools = SourceFileLoader('aitools', 'files/ai/aitools/aitools').load_module()
        data = aitools.load_config()
    except Exception:
        pass

data['default_commit_backend'] = '${selected_backend}'
commit = data.setdefault('commit', {})
commit['default_backend'] = '${selected_backend}'
commit['model'] = '${selected_model}' or None
commit['effort'] = '${selected_effort}'
commit['default_template'] = '${selected_template}'
commit['default_stage_all'] = (${selected_stage_all} == True)

daemon = data.setdefault('daemon', {})
auto_wake = daemon.setdefault('auto_wake', {})
auto_wake['enabled'] = (${selected_auto_wake} == True)
auto_wake['time'] = '${selected_wake_time}'
auto_wake['auto_resuspend'] = (${selected_resuspend} == True)

with open(config_path, 'w', encoding='utf-8') as f:
    json.dump(data, f, indent=2)

print('Persisted successfully to ' + config_path)
"

	WIZARD_WRITTEN_ENV+=(
		"DEFAULT_COMMIT_BACKEND='${selected_backend}'"
		"COMMIT_MODEL='${selected_model:-default}'"
		"COMMIT_EFFORT='${selected_effort}'"
		"COMMIT_TEMPLATE='${selected_template}'"
		"COMMIT_STAGE_ALL='${selected_stage_all}'"
		"AUTO_WAKE_ENABLED='${selected_auto_wake}'"
	)

	wizard_finish "aitools configuration completed successfully!"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
	run_aitools_wizard
fi
