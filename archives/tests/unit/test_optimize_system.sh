#!/usr/bin/env bash
# ==============================================================================
# tests/unit/test_optimize_system.sh
# Unit tests for utils/optimize_system.sh hardware profiler, Lynis security audit,
# actionable report distillation & security guardrail engine.
# ==============================================================================

# shellcheck source=/dev/null
source "$(dirname "${BASH_SOURCE[0]}")/../test_helper.sh"

echo "  Running System Optimizer & Security Guardrail Unit Tests..."

setup_sandbox
trap teardown_sandbox EXIT

OPTIMIZER="${REPO_ROOT}/utils/optimize_system.sh"
AITOOLS="${REPO_ROOT}/files/ai/aitools/aitools"

# Test 1: Executable exists and has execute permissions
if [[ -f "$OPTIMIZER" && -x "$OPTIMIZER" ]]; then
	pass "utils/optimize_system.sh exists and is marked executable"
else
	fail "Missing or non-executable optimizer at: $OPTIMIZER"
fi

# Test 2: Bash syntax validation
bash -n "$OPTIMIZER"
assert_exit_code 0 "$?" "bash -n validation of optimize_system.sh" &&
	pass "optimize_system.sh passes bash syntax validation"

# Test 3: ShellCheck static analysis
if command -v shellcheck >/dev/null 2>&1; then
	shellcheck -e SC1091 -e SC1090 "$OPTIMIZER"
	assert_exit_code 0 "$?" "ShellCheck lint of optimize_system.sh" &&
		pass "optimize_system.sh passes ShellCheck with zero warnings"
fi

# Test 4: Help interface
help_out="$("$OPTIMIZER" --help 2>&1)"
assert_contains "$help_out" "System Hardware & Performance Optimizer" "CLI help header" &&
	assert_contains "$help_out" "--wizard" "CLI help contains wizard flag" &&
	assert_contains "$help_out" "--dry-run" "CLI help contains dry-run flag" &&
	assert_contains "$help_out" "--validate" "CLI help contains validate flag" &&
	pass "optimize_system.sh --help exposes comprehensive CLI manual"

# Test 5: Hardware & security audit distillation (--audit-only --skip-lynis)
audit_out="$("$OPTIMIZER" --audit-only --skip-lynis 2>&1)"
assert_contains "$audit_out" "System Hardware, Performance & Security Audit Digest" "Audit header" &&
	assert_contains "$audit_out" "Platform:" "Audit contains platform details" &&
	assert_contains "$audit_out" "CPU Model:" "Audit contains processor model" &&
	assert_contains "$audit_out" "Total RAM:" "Audit contains memory details" &&
	assert_contains "$audit_out" "Security & Hardening Context" "Audit contains security section" &&
	pass "optimize_system.sh --audit-only distills structured high-signal Markdown report"

# Test 6: Security Guardrail: Allows clean and safe sysctl tuning script
safe_script="${TEST_SANDBOX}/safe_tune.sh"
cat << 'EOF' > "$safe_script"
#!/usr/bin/env bash
set -euo pipefail
# WHY: On SSD NVMe systems, swappiness 15 prevents page thrashing
if [[ -d /etc/sysctl.d ]]; then
    echo "vm.swappiness=15" > /etc/sysctl.d/99-test-opt.conf
    echo "fs.inotify.max_user_watches=524288" >> /etc/sysctl.d/99-test-opt.conf
fi
EOF

"$OPTIMIZER" --validate "$safe_script" >/dev/null 2>&1
assert_exit_code 0 "$?" "Security guardrail validation of safe script" &&
	pass "Guardrail engine successfully approves safe, non-destructive tuning script"

# Test 7: Security Guardrail: Rejects destructive rm -rf command
bad_rm_script="${TEST_SANDBOX}/bad_rm.sh"
cat << 'EOF' > "$bad_rm_script"
#!/usr/bin/env bash
set -euo pipefail
rm -rf /tmp/some_cache
EOF

bad_rm_out="$("$OPTIMIZER" --validate "$bad_rm_script" 2>&1 || true)"
assert_contains "$bad_rm_out" "SECURITY GUARDRAIL TRIGGERED" "Security guardrail output for rm -rf" &&
	pass "Guardrail engine strictly blocks recursive deletion commands (rm -rf)"

# Test 8: Security Guardrail: Rejects partition and filesystem formatting commands (mkfs, dd, shred)
bad_fmt_script="${TEST_SANDBOX}/bad_fmt.sh"
cat << 'EOF' > "$bad_fmt_script"
#!/usr/bin/env bash
set -euo pipefail
mkfs.ext4 /dev/sdb1
EOF

bad_fmt_out="$("$OPTIMIZER" --validate "$bad_fmt_script" 2>&1 || true)"
assert_contains "$bad_fmt_out" "SECURITY GUARDRAIL TRIGGERED" "Security guardrail output for mkfs" &&
	pass "Guardrail engine strictly blocks partition and filesystem formatters (mkfs)"

# Test 9: Security Guardrail: Rejects disabling firewall or security policies (ufw disable)
bad_fw_script="${TEST_SANDBOX}/bad_fw.sh"
cat << 'EOF' > "$bad_fw_script"
#!/usr/bin/env bash
set -euo pipefail
ufw disable
EOF

bad_fw_out="$("$OPTIMIZER" --validate "$bad_fw_script" 2>&1 || true)"
assert_contains "$bad_fw_out" "SECURITY GUARDRAIL TRIGGERED" "Security guardrail output for ufw disable" &&
	pass "Guardrail engine strictly blocks disabling system firewall (ufw disable)"

# Test 10: Security Guardrail: Rejects tampering with /etc/shadow or /etc/passwd
bad_pwd_script="${TEST_SANDBOX}/bad_pwd.sh"
cat << 'EOF' > "$bad_pwd_script"
#!/usr/bin/env bash
set -euo pipefail
echo "root::0:0:root:/root:/bin/bash" > /etc/passwd
EOF

bad_pwd_out="$("$OPTIMIZER" --validate "$bad_pwd_script" 2>&1 || true)"
assert_contains "$bad_pwd_out" "SECURITY GUARDRAIL TRIGGERED" "Security guardrail output for /etc/passwd tampering" &&
	pass "Guardrail engine strictly blocks credential database tampering (/etc/passwd)"

# Test 11: aitools optimize subcommand integration
if [[ -f "$AITOOLS" && -x "$AITOOLS" ]]; then
	aitools_opt_help="$("$AITOOLS" optimize --help 2>&1)"
	assert_contains "$aitools_opt_help" "optimize" "aitools optimize help output" &&
		assert_contains "$aitools_opt_help" "--audit-only" "aitools optimize audit-only flag" &&
		pass "aitools optimize subcommand exposes comprehensive CLI help interface"

	aitools_audit_out="$("$AITOOLS" optimize --audit-only --skip-lynis 2>&1)"
	assert_contains "$aitools_audit_out" "System Hardware, Performance & Security Audit Digest" "aitools delegation" &&
		pass "aitools optimize --audit-only cleanly delegates to optimize_system.sh engine"
fi
