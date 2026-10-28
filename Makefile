# ==============================================================================
# Makefile
# Developer & Operational Automation Interface for arturonavax/env.
# ==============================================================================

.DEFAULT_GOAL := help
.PHONY: help test test-unit test-integration lint check wizard plan dry-run uninstall-plan backup restore diff sync clean

help:
	@echo "arturonavax/env — Development & Operational Interface"
	@echo ""
	@echo "Available Targets:"
	@echo "  make plan              Preview installation plan, packages, and touched files (dry-run)"
	@echo "  make uninstall-plan    Preview uninstallation plan, purged binaries, and caches"
	@echo "  make test              Run comprehensive test harness (syntax, lint, unit, integration)"
	@echo "  make test-unit         Run unit test suite only"
	@echo "  make test-integration  Run integration smoke test suite only"
	@echo "  make lint              Run syntax check (bash -n) and ShellCheck analysis"
	@echo "  make check             Run environment & hardware pre-flight check"
	@echo "  make wizard            Launch 8-stage interactive installation wizard"
	@echo "  make backup            Create timestamped configuration snapshot"
	@echo "  make restore           Interactively restore configuration from backup"
	@echo "  make diff              Inspect configuration diffs against active system"
	@echo "  make sync              Propagate repository dotfiles into active system"
	@echo "  make clean             Remove temporary test sandboxes and stale caches"
	@echo ""

plan:
	@./install.sh --plan

dry-run:
	@./install.sh --dry-run

uninstall-plan:
	@./uninstall.sh --plan

test:
	@./tests/run_tests.sh

test-unit:
	@./tests/run_tests.sh --unit

test-integration:
	@./tests/run_tests.sh --integration

lint:
	@./tests/run_tests.sh --lint

check:
	@./install.sh --check

wizard:
	@./install.sh --wizard

backup:
	@./utils/backup_config.sh

restore:
	@./utils/restore_backup.sh

diff:
	@./utils/diff_snapshots.sh

sync:
	@./utils/sync_config.sh

clean:
	@rm -rf /tmp/env_test_sandbox.* /tmp/bats_sandbox.* 2>/dev/null || true
	@echo "Cleaned temporary test sandboxes."
