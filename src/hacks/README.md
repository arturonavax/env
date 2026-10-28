# Workstation-Specific Hacks & Workarounds (`src/hacks`)

This directory centralizes specialized hardware and desktop environment workarounds that apply to specific machines, GPU profiles, or situational workflows. They are kept separated from the general development environment installer.

---

## 1. `activate_scroll_lock.sh`
- **Purpose**: Hardware LED controller for keyboards where Scroll Lock key controls the backlight (e.g., Cooler Master Devastator and similar gaming keyboards under Linux).
- **Target Setup**: Linux X11 systems via `xset led named "Scroll Lock"`.

---

> **Note on Quota Priming & Auto-Wake**:
> The legacy standalone quota daemon has been evolved and promoted into **`aitools`** (`files/ai/aitools/`), an extensible multi-model, multi-agent AI suite supporting `agy`, `claude`, `codex`, `opencode`, `pi`, and `llm`. See [files/ai/aitools/README.md](../../files/ai/aitools/README.md).
