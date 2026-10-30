# Workstation-Specific Hacks & Workarounds (`src/hacks`)

> **Note**: Declarative system optimizations and hardware hacks are now centrally declared in [`manifests/hacks.yaml`](../../manifests/hacks.yaml) and bundled in [`manifests/hacks/`](../../manifests/hacks/). They are managed directly via `anx hacks [detect|apply|info]`.

This directory preserves compatibility shims for external execution scripts:

---

## 1. `activate_scroll_lock.sh`
- **Purpose**: Hardware LED controller for keyboards where Scroll Lock key controls the backlight (e.g., Cooler Master Devastator and similar gaming keyboards under Linux).
- **Target Setup**: Linux X11 systems via `xset led named "Scroll Lock"`. Canonical implementation lives in `manifests/hacks/x11-scroll-lock-led/`.

---

> **Note on Quota Priming & Auto-Wake**:
> The legacy standalone quota daemon has been evolved and promoted into **`aitools`** / **`anx tools ai`** (`files/ai/aitools/`), an extensible multi-model, multi-agent AI suite supporting `agy`, `claude`, `codex`, `opencode`, `pi`, and `llm`. See [files/ai/aitools/README.md](../../files/ai/aitools/README.md).
