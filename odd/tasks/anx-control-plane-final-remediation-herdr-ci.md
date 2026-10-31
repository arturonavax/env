# ANX Control Plane Final Remediation Board (Herdr, Windows CI, Dotfile Security)

**ID:** `anx-control-plane-final-remediation-herdr-ci`  
**Base HEAD:** `9fe41da2569588a526ccfb20d991e82bc8506444`  
**Status:** Completed  
**Normative Reference:** `odd/ANX_ODD_Final_Remediation_Herdr_CI_2026-10-30.md`, `odd/specs/anx/control-plane-v2.md`  

---

## 1. Local Diagnostics & Herdr Runtime Isolation (R02, HS09, HS10, HS12)

- [x] **Local Inspection:** Verified `~/.config/herdr` was previously a directory symlink -> `files/herdr`.
  - Evidence: `readlink -f ~/.config/herdr` returned `<repo>/files/herdr`.
  - Impact: Herdr wrote runtime artifacts (`herdr.sock`, `session.json`, `session-snapshots/`, logs) directly into repo working tree.
  - Action: Decoupled directory symlink. Converted `~/.config/herdr` into an independent real directory, preserved session snapshots in user home, and purged all residual runtime files from `files/herdr/`.
  - Policy Enforced: Strict default-deny in `.gitignore`, restricted sync to single vetted `config.toml`, and eliminated whole-tree staging. `files/herdr/` now exclusively tracks `config.toml`.

---

## 2. Priority 0 Tasks (R00 - R09)

- [x] **R00:** Baseline reconciliation and ODD task matrix open.
- [x] **R01:** Fix 8 Windows test failure families:
  - [x] `internal/ai/gitflow:TestClipboardOSC52FallbackAndBinary`: Cross-platform clipboard runner mock.
  - [x] `internal/ai/harness`: Replaced bash fake scripts with native mock runner (`mockTestRunner`).
  - [x] `internal/extension`: Replaced `mock_plugin.sh` with native Go JSON-RPC mock executable (`internal/extension/testdata/mock_plugin/main.go`).
  - [x] `internal/hacks`: Eliminated `:` in NTFS test paths (`0000:01:00.0` -> `0000_01_00.0`).
  - [x] `internal/ops:TestRebuilderDryRunAndMockExecution`: Cross-platform executable mock (`internal/ops/testdata/mock_tool/main.go`).
  - [x] `internal/ops:TestOptimizerAuditSynthesizeAndGuardrails`: Platform-aware optimizer mock runner.
  - [x] `internal/providers:TestMultiStrategyDispatcherGitHubTarballFallbackToScript`: Cross-platform binary fixture with `.cmd`/`.bat` support.
  - [x] `internal/tools:TestToolRunnerExternal`: Portable LookPath mock and Windows `.exe`/`.cmd` extension handling.
- [x] **R03:** Default-deny allowlist in `.gitignore` for `files/herdr/**` except `!/files/herdr/config.toml`.
- [x] **R04:** Capture typed ChangeSet, exact staging `git add -- <paths>`, eliminate `git add files/`.
- [x] **R05:** Universal `--dry-run` guarantee (zero git mutations, zero disk writes).
- [x] **R06:** `anx self update` verification and mock server testing.
- [x] **R07:** SQLite state store with WAL, transactional migrations, and mutex lock around disk writes.
- [x] **R08:** PlanSpec immutability, digest validation, and stale plan rejection.
- [x] **R09:** Extension host launch trust policy and permission model.

---

## 3. Negative Security Tests (HS01 - HS12)

- [x] `HS01`: Herdr config capturable, unapproved files never captured.
- [x] `HS02`: `session.json`, sockets, snapshots never enter commits even with `--all`.
- [x] `HS03`: `git check-ignore` confirms deny-all Herdr except `config.toml`.
- [x] `HS04`: `anx capture --all --dry-run --commit --push` causes 0 git mutations, 0 network, 0 writes.
- [x] `HS05`: Exact changeset staging, no collateral staging of pre-existing modified files.
- [x] `HS06`: Path traversal and symlink evasion blocked.
- [x] `HS07`: Malformed/empty manifest fail-closed.
- [x] `HS08`: Secret scanning blocks commit on sensitive tokens.
- [x] `HS09`: Active sockets/sessions not deleted or moved.
- [x] `HS10`: Clean installation uses single file template, not directory symlink.
- [x] `HS11`: Cross-platform capture parity.
- [x] `HS12`: Real diagnosis documented without speculation.

---

## 4. Priority 1 & 2 Tasks (R10 - R25, R99)

- [x] **R10:** Scheduler DAG dependency resolution.
- [x] **R11:** Durable jobs and events.
- [x] **R12:** Manifest versioning & provenance.
- [x] **R13:** Semantic JSONC/TOML 3-way merge.
- [x] **R14:** Safe alias and ownership-guarded uninstallation.
- [x] **R15:** Fail-closed `Observe` and real provider error reporting.
- [x] **R16:** Capability wrappers with flag validation.
- [x] **R17:** Core ports/adapters for CLI/TUI.
- [x] **R18:** Go >= 1.27.3 floor & Node 26 LTS across all docs and scripts.
- [x] **R19:** Dependency updates and GitHub Actions hardening.
- [x] **R20:** Mutex protection in `state.Store` and CAS SHA verification on reuse.
- [x] **R21:** Secret scanning & privacy protection.
- [x] **R22:** Multi-platform testing (Linux, macOS, Windows).
- [x] **R23:** Branch protection & CI requirements.
- [x] **R24:** Documentation and completions regeneration.
- [x] **R25:** Future-ready API v1 DTOs.
- [x] **R99:** Final verification gate (4/4 test runner suites passed with zero data races and zero lint errors).

---

## 5. Security & Power Management Enhancements (New Deliverables)

- [x] **S01: Dedicated Security Audit & Remediation Tool (`anx audit`):**
  - Multi-level system security auditing (`Critical`, `High`, `Medium`, `Low`, `Info`) across network, authentication, permissions, kernel sysctls, and packages.
  - Profile support: `workstation`, `server`, and `minimal` security baselines.
  - Standalone idempotent remediation script generation with 12 security guardrails.
  - Optional `--ai` LLM script synthesis using configured harnesses.
  - Interactive Bubble Tea v2 TUI tab with profile switching ('p') and direct remediation script generation ('s').
- [x] **S02: Universal `--script` Mode:**
  - Standardized `--script` flag across all `anx` tools and commands (`apply`, `sync`, `optimize`, `audit`, `rebuild`, `capture`, `cleanup`/`prune`) printing executable Bash script previews of all actions before execution.
- [x] **S03: Autonomous Quota Wakeup & RTC Wake Fix for Suspended Laptops:**
  - Replaced local time with UTC in `rtcwake -m no -u -t <epoch>`.
  - Added direct sysfs fallback to `/sys/class/rtc/rtc0/wakealarm`.
  - Enhanced sleep resume handler in `HandleDBusLine`: automatically triggers missed wake cycles for `agy` upon system resume even if suspend exceeded the previous 15-minute window.
