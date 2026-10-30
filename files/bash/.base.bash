# ==============================================================================
# ~/.base.bash — arturonavax/env Bash Core Environment
# Derived from zsh single-source-of-truth for cross-shell compatibility
# ==============================================================================

# 1. Environment & Development Toolpaths
if [ -f "$HOME/.tools.sh" ]; then
    . "$HOME/.tools.sh"
fi

# 2. LSCOLORS / Colors
if [ -f "$HOME/.lscolors.sh" ]; then
    . "$HOME/.lscolors.sh"
fi

# 3. Personal Tools & Local Binaries
if [ -d "$HOME/.local/share/anx/bin" ] && [[ ":$PATH:" != *":$HOME/.local/share/anx/bin:"* ]]; then
    export PATH="$HOME/.local/share/anx/bin:$PATH"
fi
if [ -d "$HOME/.local/bin" ] && [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    export PATH="$HOME/.local/bin:$PATH"
fi

# 4. FZF Shell Integration
if command -v fzf >/dev/null 2>&1; then
    eval "$(fzf --bash 2>/dev/null)"
fi

# 5. Atuin SQLite History
if command -v atuin >/dev/null 2>&1; then
    eval "$(atuin init bash 2>/dev/null)"
fi

# 6. Starship Prompt (Compact Lean Theme)
if command -v starship >/dev/null 2>&1; then
    eval "$(starship init bash 2>/dev/null)"
fi

# 7. Common Core Aliases (Matching Zsh)
alias ll='eza -lh --group-directories-first --icons'
alias la='eza -lah --group-directories-first --icons'
alias l='eza -lah --group-directories-first --icons'
alias cat='bat --paging=never'
alias diff='difft'
