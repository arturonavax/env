# ==============================================================================
# ~/.config/nushell/env.nu — arturonavax/env Environment Variables
# ==============================================================================

# >>> arturonavax/env >>>
$env.PATH = ($env.PATH | split row (char esep) | prepend [
    $"($env.HOME)/.local/share/anx/bin",
    $"($env.HOME)/.local/bin"
] | uniq)
# <<< arturonavax/env <<<
