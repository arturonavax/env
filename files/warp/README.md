# Warp Terminal Compatibility & Architectural Notice

> [!WARNING]
> **Limited Feature Compatibility with Warp Terminal**
>
> Warp enforces its own proprietary command-block execution model, cloud telemetry, and internal shell integration. Because of Warp's closed design:
>
> 1. **Starship Prompt is Disabled / Overridden**: Warp ignores or breaks custom prompt rendering hooks (`starship init`) in favor of its native block headers.
> 2. **Shell FZF Interactive Widgets are Incompatible**: Warp intercepts keystrokes for its own autosuggestions and completions. FZF `Ctrl-R` and tab completions conflict with Warp's event pipeline.
> 3. **Herdr Agent Multiplexer Cannot Manage Warp Blocks**: Warp does not expose native PTY socket supervision to multiplexers like Herdr.
>
> **Recommendation**: For the complete feature suite (Starship, Atuin SQLite history, FZF tab completion, Herdr workspace supervision), use **Ghostty** (recommended), **Kitty**, **WezTerm**, or **Alacritty**.
