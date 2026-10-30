# ==============================================================================
# ~/.config/fish/config.fish — arturonavax/env Fish Core Environment
# ==============================================================================

# >>> arturonavax/env >>>
# Personal Tools & Path Isolation
if test -d $HOME/.local/share/anx/bin
    set -gx PATH $HOME/.local/share/anx/bin $PATH
end
if test -d $HOME/.local/bin
    set -gx PATH $HOME/.local/bin $PATH
end

# Core Aliases matching zsh/bash
if type -q eza
    alias ll="eza -lh --group-directories-first --icons"
    alias la="eza -lah --group-directories-first --icons"
    alias l="eza -lah --group-directories-first --icons"
end
if type -q bat
    alias cat="bat --paging=never"
end
if type -q difft
    alias diff="difft"
end

# Interactive Integrations
if status is-interactive
    # Atuin SQLite History
    if type -q atuin
        atuin init fish | source
    end

    # FZF Keybindings
    if type -q fzf
        fzf --fish | source
    end

    # Starship Prompt
    if type -q starship
        starship init fish | source
    end
end
# <<< arturonavax/env <<<
