#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_plan.sh
# Unit tests for installation plan, uninstallation plan, and filesystem manifests.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/plan.sh"

echo "  Running Installation & Teardown Plan Tests..."

# Test 1: Full Installation Plan Output
plan_all_output="$(show_install_plan all)"
assert_contains "$plan_all_output" "Installation Plan & Manifest Preview" "Full plan contains title banner" &&
	assert_contains "$plan_all_output" "Pre-Installation Safeguards & Backups" "Plan contains backup safeguards" &&
	assert_contains "$plan_all_output" "Packages to be Installed via Native Manager" "Plan contains package translations" &&
	assert_contains "$plan_all_output" "Standalone Binaries & High-Performance CLI Tools" "Plan contains CLI tools" &&
	assert_contains "$plan_all_output" "File System Interactions" "Plan contains file interactions" &&
	assert_contains "$plan_all_output" "SMART-APPEND" "Plan documents smart-append-first marker blocks" &&
	assert_contains "$plan_all_output" "PRESERVED" "Plan documents preserved user files" &&
	pass "show_install_plan all generated complete manifest with file interactions"

# Test 2: Scoped Subsystem Plan Output (Terminal Only)
plan_term_output="$(show_install_plan terminal)"
assert_contains "$plan_term_output" "Ghostty" "Terminal plan contains Ghostty" &&
	assert_contains "$plan_term_output" "Herdr" "Terminal plan contains Herdr" &&
	assert_contains "$plan_term_output" "Atuin" "Terminal plan contains Atuin" &&
	assert_contains "$plan_term_output" "zsh-autosuggestions" "Terminal plan contains zsh plugins" &&
	pass "show_install_plan terminal generated scoped subsystem plan"

# Test 3: Uninstallation Plan Output
uninstall_plan_output="$(show_uninstall_plan)"
assert_contains "$uninstall_plan_output" "Uninstallation Plan & Teardown Preview" "Uninstall plan contains title banner" &&
	assert_contains "$uninstall_plan_output" "Packages Removed via Native Manager" "Uninstall plan lists removed packages" &&
	assert_contains "$uninstall_plan_output" "System & Local Binaries Purged" "Uninstall plan lists purged binaries" &&
	assert_contains "$uninstall_plan_output" "Desktop Entries, Icons & Systemd Services" "Uninstall plan lists desktop services" &&
	assert_contains "$uninstall_plan_output" "Configuration Directories, Local States & Caches Purged" "Uninstall plan lists purged configs" &&
	assert_contains "$uninstall_plan_output" "Files & Assets Preserved (NEVER DELETED)" "Uninstall plan explicitly confirms preserved assets" &&
	assert_contains "$uninstall_plan_output" ".arturonavax-env-backups" "Uninstall plan confirms backup folder preserved" &&
	pass "show_uninstall_plan generated teardown manifest with purged and preserved assets"
