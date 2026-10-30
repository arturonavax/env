# ==============================================================================
# Makefile
# Developer & Operational Automation Interface for arturonavax/env.
# ==============================================================================

.DEFAULT_GOAL := help
.PHONY: help build install test test-go test-e2e test-unit test-integration lint check wizard plan dry-run uninstall-plan backup restore diff sync clean rebuild rebuild-tools optimize optimize-apply anx-plan anx-status anx-doctor

VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "1.0.0")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
DATE    ?= $(shell date -u "+%Y-%m-%d")
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)

build:
	@mkdir -p bin
	@go build -ldflags "$(LDFLAGS)" -o bin/anx ./cmd/anx
	@ln -sf anx bin/aitools
	@echo "Built bin/anx and bin/aitools (multicall symlink) successfully."

install:
	@./scripts/bootstrap_anx.sh apply

test-go:
	@go test -v -race ./...

test-e2e:
	@./scripts/test_e2e_docker.sh all

anx-plan:
	@go run -ldflags "$(LDFLAGS)" ./cmd/anx plan

anx-status:
	@go run -ldflags "$(LDFLAGS)" ./cmd/anx status

anx-doctor:
	@go run -ldflags "$(LDFLAGS)" ./cmd/anx doctor

help:
	@echo "arturonavax/env — Development & Operational Interface"
	@echo ""
	@echo "Available Targets:"
	@echo "  make build             Compile anx and aitools binaries into bin/"
	@echo "  make install           Bootstrap Go and apply workstation state via anx"
	@echo "  make test-go           Run full Go unit & race test suite"
	@echo "  make test              Run comprehensive test harness (syntax, lint, unit, integration)"
	@echo "  make anx-plan          Run anx execution planner"
	@echo "  make anx-status        Inspect anx platform and package status"
	@echo "  make anx-doctor        Run anx diagnostic health checks"
	@echo "  make plan              Preview installation plan, packages, and touched files (dry-run)"
	@echo "  make uninstall-plan    Preview uninstallation plan, purged binaries, and caches"
	@echo "  make lint              Run syntax check (bash -n) and ShellCheck analysis"
	@echo "  make wizard            Launch interactive installation wizard"
	@echo "  make clean             Remove temporary test sandboxes and stale caches"
	@echo ""

optimize:
	@./utils/optimize_system.sh --dry-run

optimize-apply:
	@./utils/optimize_system.sh --apply

rebuild:
	@./utils/rebuild_tools.sh

rebuild-tools: rebuild

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
	@rm -rf bin/ /tmp/env_test_sandbox.* /tmp/bats_sandbox.* 2>/dev/null || true
	@echo "Cleaned temporary test sandboxes and bin/."
