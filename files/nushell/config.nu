# ==============================================================================
# ~/.config/nushell/config.nu — arturonavax/env Nushell Environment
# ==============================================================================

# >>> arturonavax/env >>>
$env.config = {
    show_banner: false
}

# Core Aliases
alias ll = eza -lh --group-directories-first --icons
alias la = eza -lah --group-directories-first --icons
alias cat = bat --paging=never
alias diff = difft

# Shell Integrations
if (which starship | is-not-empty) {
    mkdir ~/.cache/starship
    starship init nu | save -f ~/.cache/starship/init.nu
    source ~/.cache/starship/init.nu
}

if (which atuin | is-not-empty) {
    mkdir ~/.cache/atuin
    atuin init nu | save -f ~/.cache/atuin/init.nu
    source ~/.cache/atuin/init.nu
}
# <<< arturonavax/env <<<
