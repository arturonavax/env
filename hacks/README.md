# Declarative Workstation Hacks (`hacks/`)

`hacks/` is the structured, opt-in home for machine-, GPU-, display-server-, and peripheral-specific workarounds managed by `anx hacks`.

Unlike core dotfiles and cross-platform CLI toolchains, workstation hacks target specific hardware topologies (for example, NVIDIA Kepler GPUs running `nouveau` on Xfce, or X11 keyboard LED quirks) and are **never applied implicitly** unless explicitly selected or invoked via `anx hacks apply <id>`.

---

## Directory Layout

Each built-in workstation hack lives in its own subdirectory under `hacks/<hack-id>/`:

```text
hacks/
├── embed.go                              # Go embed.FS bundling */hack.yaml and scripts
├── README.md                             # Authoring guide for declarative workstation hacks
├── dell-precision-m6800/
│   ├── hack.yaml                         # Hardware match predicates, backup paths & steps
│   ├── apply_hacks.sh                    # Idempotent workstation tuning script
│   └── README.md                         # Deep-dive root cause analysis
└── x11-scroll-lock-led/
    ├── hack.yaml                         # Linux + X11 display server matcher
    └── activate_scroll_lock.sh           # Defensive xset Scroll Lock LED controller
```

### Dynamic Overlay Precedence

`anx` loads hack definitions in three layers without requiring Go recompilation:

1. **Embedded Defaults** (`hacks.EmbeddedFS`): Bundled into the `anx` binary for standalone use.
2. **Repository Checkout** (`<repo>/hacks/*/hack.yaml`): Live workspace edits override embedded profiles with the same `id`.
3. **User Drop-In Directory** (`~/.config/anx/hacks.d/*.yaml`): Personal or machine-local profiles overlay both embedded and repository definitions.

---

## Adding a New Workstation Hack (Zero Go Changes)

1. Create a folder `hacks/<your-hack-id>/` (or a YAML file in `~/.config/anx/hacks.d/<your-hack-id>.yaml`).
2. Add an idempotent shell script (using `#!/usr/bin/env bash`, `set -euo pipefail`, and `$HOME` rather than hardcoded usernames).
3. Declare `hack.yaml` following the `domain.HackProfile` schema:

```yaml
id: thinkpad-x1-throttled
name: ThinkPad X1 Carbon Thermal Throttle Workaround
category: workstation
description: Adjusts power limits and sleep hooks for Intel thermal throttling on Linux.
safe_by_opt_in: true
match:
  os:
    - linux
  dmi_vendor: LENOVO
  dmi_product: 20KH
  desktop_env:
    - gnome
    - xfce
  display_server:
    - x11
    - wayland
backup_paths:
  - "~/.config/environment.d"
steps:
  - id: apply-thermal-profile
    name: Configure Thermal Profile
    description: Deploys user environment overrides and verifies thermald settings.
    script: apply.sh
    touches:
      - "~/.config/environment.d/90-thinkpad.conf"
    privileged: false
```

---

## Safety Guarantees

Every hack executed through `anx hacks apply` passes through four mandatory safeguards:

1. **Opt-In Hardware Matching**: `anx hacks detect` probes DMI (`sys_vendor`, `product_name`), PCI GPU models/drivers, OS/distro, `XDG_CURRENT_DESKTOP`, and display server (`x11` vs `wayland`) to recommend matching profiles without auto-mutating unrelated hosts.
2. **Static Guardrail Scanning**: Every referenced script is scanned with `security.ValidateScriptGuardrails` before any step runs. Destructive patterns (`rm -rf /`, `mkfs`, `dd` to block devices, firewall disabling, `/etc/shadow` redirects) abort execution immediately.
3. **Pre-Hack SHA-256 Snapshot**: All existing files and directories listed in `backup_paths` are snapshotted into `<stateDir>/backups/<timestamp>/payload/...` with SHA-256 digests prior to execution.
4. **Audit & State Recording**: Applied hack IDs and step lists are persisted atomically in `state.json` (`applied_hacks`).
