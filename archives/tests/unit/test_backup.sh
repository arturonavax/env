#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_backup.sh
# Unit tests for cross-platform backup creation, manifest verification, rotation,
# and rollback restoration.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"
# shellcheck source=/dev/null
source "${REPO_ROOT}/src/lib/backup.sh"

echo "  Running Backup & Rollback Engine Tests..."

setup_sandbox
trap teardown_sandbox EXIT

export BACKUP_STORAGE_DIR="${TEST_SANDBOX}/backups"
mkdir -p "$BACKUP_STORAGE_DIR"

# Test 1: Snapshot Creation and Manifest Generation
snapshot_path="$(backup_create)"

assert_file_exists "$snapshot_path" "Snapshot folder created" &&
	assert_file_exists "${snapshot_path}/manifest.txt" "Manifest file created in snapshot" &&
	assert_file_contains "${snapshot_path}/manifest.txt" "# arturonavax/env Backup Manifest" "Manifest contains header" &&
	pass "backup_create generated timestamped snapshot with manifest"

# Test 2: Portable Size Calculation
size_kb="$(_get_dir_size_kb "$snapshot_path")"
if ((size_kb >= 0)); then
	pass "_get_dir_size_kb calculated directory size: ${size_kb} KB"
else
	fail "_get_dir_size_kb returned negative size"
fi

# Test 3: Snapshot Rotation Quota Enforcement
# Create dummy older snapshots
snap1="${BACKUP_STORAGE_DIR}/2025-01-01_10-00-00"
snap2="${BACKUP_STORAGE_DIR}/2025-01-02_10-00-00"
mkdir -p "$snap1" "$snap2"
# Write 1MB into each dummy snapshot
dd if=/dev/zero of="${snap1}/large.bin" bs=1024 count=1024 2>/dev/null
dd if=/dev/zero of="${snap2}/large.bin" bs=1024 count=1024 2>/dev/null

# Set maximum quota to 1.5MB (1536 KB) so the oldest (snap1) must be rotated out
export BACKUP_MAX_SIZE_KB=1536
_rotate_backups

if [[ ! -d "$snap1" ]]; then
	pass "_rotate_backups successfully purged oldest snapshot exceeding quota"
else
	fail "_rotate_backups failed to purge oldest snapshot"
fi

if [[ -d "$snap2" ]]; then
	pass "_rotate_backups preserved newer snapshot within quota"
else
	fail "_rotate_backups prematurely purged newer snapshot"
fi

# Test 4: Rollback / Restore Simulation
restore_test_dir="${TEST_SANDBOX}/restore_target"
mkdir -p "${restore_test_dir}/config"
echo "restored_content=true" >"${restore_test_dir}/.zshrc"

dummy_snapshot="${BACKUP_STORAGE_DIR}/2025-01-03_12-00-00"
mkdir -p "$dummy_snapshot"
echo "restored_payload=true" >"${dummy_snapshot}/.zshrc"

HOME="$restore_test_dir" backup_restore "$dummy_snapshot"

assert_file_contains "${restore_test_dir}/.zshrc" "restored_payload=true" "Restored file from snapshot" &&
	pass "backup_restore restored configuration files cleanly"
