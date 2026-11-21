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
def --wrapped safe-suspend [...args] {
    if (which anx | is-not-empty) {
        ^anx sleep ...$args
    } else {
        let repo_root = ($env.REPO_ROOT? | default $"($env.HOME)/repos/github.com/arturonavax/env")
        let script_path = $"($repo_root)/scripts/safe_suspend.sh"
        if ($script_path | path exists) {
            ^bash $script_path ...$args
        } else {
            error make { msg: "safe-suspend command not found" }
        }
    }
}
alias suspend-safe = safe-suspend
alias sleep-safe = safe-suspend

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
