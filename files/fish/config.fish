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
function safe-suspend
    if type -q anx
        anx sleep $argv
    else if test -x "$HOME/.local/bin/anx"
        "$HOME/.local/bin/anx" sleep $argv
    else
        set -l repo_root (test -n "$REPO_ROOT"; and echo "$REPO_ROOT"; or echo "$HOME/repos/github.com/arturonavax/env")
        set -l script_path "$repo_root/scripts/safe_suspend.sh"
        if test -f "$script_path"
            bash "$script_path" $argv
        else
            echo "safe-suspend command not found" >&2
            return 1
        end
    end
end
alias suspend-safe="safe-suspend"
alias sleep-safe="safe-suspend"

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
