#!/bin/zsh
[[ "$0" != *"zsh"* ]] && return

bindkey -e

# PATH Basics
[[ ":$PATH:" != *":/usr/local/bin:"* ]] && export PATH="$PATH:/usr/local/bin"
[[ ":$PATH:" != *":$HOME/.local/bin:"* ]] && export PATH="$HOME/.local/bin:$PATH"
[[ -d "$HOME/.fzf/bin" && ":$PATH:" != *":$HOME/.fzf/bin:"* ]] && export PATH="$HOME/.fzf/bin:$PATH"
[[ -d "$HOME/.atuin/bin" && ":$PATH:" != *":$HOME/.atuin/bin:"* ]] && export PATH="$HOME/.atuin/bin:$PATH"

# Linux - GTK Renderer (Estabilidad y prevención de fugas de texturas en GTK 4.14 / Wayland)
if [[ "$(uname -s)" == "Linux" ]]; then
    export GSK_RENDERER=gl
fi

# MacOS - Homebrew
if [[ "$(uname -s)" == "Darwin" ]]; then
    brewbin="/usr/local/bin/brew"
    [[ ! -f "$brewbin" ]] && brewbin="/opt/homebrew/bin/brew" # For Apple Silicon
    [[ -f "$brewbin" ]] && eval "$("$brewbin" shellenv)"
fi

# EDITOR
if [[ "$(command -v nvim)" != "" ]]; then
  export EDITOR=nvim
  export VISUAL=nvim
elif [[ "$(command -v code)" != "" ]]; then
  export EDITOR=code
  export VISUAL=code
elif [[ "$(command -v vim)" != "" ]]; then
  export EDITOR=vim
  export VISUAL=vim
fi

## ls colors
[[ -f ~/.lscolors.sh ]] && source ~/.lscolors.sh

## zsh history
export \
    HISTFILE="$HOME/.zsh_history" \
    HISTFILESIZE=1000000 \
    HISTSIZE=1000000 \
    SAVEHIST=1000000

setopt INC_APPEND_HISTORY \
    SHARE_HISTORY \
    EXTENDED_HISTORY \
    HIST_FIND_NO_DUPS \
    HIST_REDUCE_BLANKS \
    HIST_VERIFY

## utf8
export \
    LC_ALL=en_US.UTF-8 \
    LANG=en_US.UTF-8 \
    LANGUAGE=en_US.UTF-8

# ==============================================================================
# 1. MOTOR DE COMPLETION (COMPINIT OPTIMIZADO)
# ==============================================================================
autoload -Uz compinit
if [[ -n ${ZDOTDIR:-$HOME}/.zcompdump(#qN.mh+24) ]]; then
    compinit
else
    compinit -C
fi

# ==============================================================================
# 2. OPCIONES ZLE Y ESTILOS NATIVOS DE ZSH
# ==============================================================================
setopt AUTO_MENU
setopt COMPLETE_IN_WORD
setopt ALWAYS_TO_END
setopt EXTENDED_GLOB
setopt GLOB_COMPLETE
setopt LIST_AMBIGUOUS
unsetopt MENU_COMPLETE

# Estilos de completion nativos
zstyle ':completion:*' menu no
zstyle ':completion:*' insert-tab false
zstyle ':completion:*:descriptions' format '[%d]'
zstyle ':completion:*' list-colors ${(s.:.)LS_COLORS}
zstyle ':completion:*' matcher-list 'm:{[:lower:][:upper:]}={[:upper:][:lower:]}' 'r:|[._-]=* r:|=*' 'l:|=* r:|=*'

# Permitir navegación relativa hacia atrás con '..'
zstyle ':completion:*' special-dirs ..

# Redes
zstyle ':completion:*:(scp|rsync):*' tag-order 'hosts:-ipaddr:ip\ address hosts:-host:host files'
zstyle ':completion:*:(ssh|scp|rsync):*:hosts-host' ignored-patterns '*(.|:)*' loopback ip6-loopback localhost ip6-localhost broadcasthost
zstyle ':completion:*:(ssh|scp|rsync):*:hosts-ipaddr' ignored-patterns '^(<->.<->.<->.<->|(|::)([[:xdigit:].]##:(#c,2))##(|%*))' '127.0.0.<->' '255.255.255.255' '::1' 'fe80::*'

# ==============================================================================
# 3. FZF CORE Y GENERADORES RECURSIVOS (FD)
# ==============================================================================
if (( $+commands[fzf] )); then
    eval "$(fzf --zsh)"

    # Desacoplar fzf de Tab estándar para dar control total a fzf-tab
    bindkey '^I' expand-or-complete

    export FZF_DEFAULT_COMMAND='fd --type f --strip-cwd-prefix --hidden --follow --exclude .git'
    export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
    export FZF_ALT_C_COMMAND='fd --type d --strip-cwd-prefix --hidden --follow --exclude .git'

    export FZF_DEFAULT_OPTS="--height 45% --layout=reverse --border=rounded --inline-info --cycle \
        --preview 'if [ -d {} ]; then eza --tree --level=2 --color=always {} 2>/dev/null | head -200; else bat --style=numbers --color=always --line-range :300 {} 2>/dev/null || cat {}; fi' \
        --preview-window right:60%:wrap --bind 'ctrl-/:toggle-preview'"
    export FZF_CTRL_T_OPTS="--preview 'bat --style=numbers --color=always --line-range :300 {} 2>/dev/null || cat {}' --preview-window right:60%:wrap --bind 'ctrl-/:toggle-preview'"
    export FZF_ALT_C_OPTS="--preview 'eza --tree --level=2 --icons --color=always {} 2>/dev/null' --preview-window right:60%:wrap --bind 'ctrl-/:toggle-preview'"

    _fzf_compgen_path() { fd --type f --hidden --follow --exclude .git . "$1"; }
    _fzf_compgen_dir()  { fd --type d --hidden --follow --exclude .git . "$1"; }
fi

# ==============================================================================
# 4. FZF-TAB (SINCRONIZACIÓN DE COLOR TOTAL + RESPONSIVE)
# ==============================================================================
if [[ -f ~/.zsh/fzf-tab/fzf-tab.plugin.zsh ]] && (( $+commands[fzf] )); then
    source ~/.zsh/fzf-tab/fzf-tab.plugin.zsh

    # Paleta canónica sincronizada (10 slots deterministas para grupos)
    local -a ftb_group_palette=(
        $'\033[38;5;75m'   # 1: Azul celeste (Main porcelain)
        $'\033[38;5;221m'  # 2: Amarillo cálido (Ancillary manipulator)
        $'\033[38;5;176m'  # 3: Púrpura / Magenta (Ancillary interrogator)
        $'\033[38;5;81m'   # 4: Cyan (Plumbing manipulator)
        $'\033[38;5;204m'  # 5: Coral / Rojo (Plumbing interrogator)
        $'\033[38;5;114m'  # 6: Verde suave (Plumbing sync)
        $'\033[38;5;209m'  # 7: Naranja / Durazno (Plumbing sync helper / ancillary)
        $'\033[38;5;141m'  # 8: Lavanda / Violeta (Foreign / interacting)
        $'\033[38;5;43m'   # 9: Turquesa
        $'\033[38;5;211m'  # 10: Rosa
    )

    # Inyectar la paleta en fzf-tab para que coloree los resultados de la izquierda
    zstyle ':fzf-tab:*' group-colors $ftb_group_palette

    # Supresión de cabeceras invasivas y prefijos artificiales
    zstyle ':fzf-tab:*' show-group none
    zstyle ':fzf-tab:*' single-group none
    zstyle ':fzf-tab:*' prefix ''
    zstyle ':fzf-tab:*' default-color ''

    # Desvincular '/' para escribir rutas relativas (../) en el buscador.
    # Para descender a una carpeta o subir a '..' sin salir del modal, usa 'Ctrl+Espacio'.
    zstyle ':fzf-tab:*' continuous-trigger 'ctrl-space'

    # Flags FZF adaptativos y delimitador rígido
    zstyle ':fzf-tab:*' fzf-flags \
        '--height=~65%' \
        '--min-height=12' \
        '--layout=reverse' \
        '--border=rounded' \
        '--prompt=❯ ' \
        '--info=inline-right' \
        '--cycle' \
        '--bind=tab:down,btab:up' \
        '--preview-window=right,50%,wrap,border-left,<95(down,40%,wrap,border-top),<55(hidden),<18(hidden)' \
        '--bind=ctrl-/:toggle-preview' \
        '--delimiter= +-- +' \
        '--with-nth=1' \
        '--nth=1..'

    # Previews específicos por comando
    zstyle ':fzf-tab:complete:(kill|pkill):*' fzf-preview 'ps --pid=$word -o cmd --no-headers -w -w'
    zstyle ':fzf-tab:complete:(-command-|-parameter-|-brace-parameter-|export|unset|expand):*' fzf-preview 'echo ${(P)word}'

    # Previsualizador contextual con sincronización cromática exacta
    zstyle ':fzf-tab:complete:*:*' fzf-preview \
        'if [[ -n "$realpath" && -e "$realpath" && "$word" != -* ]]; then
            if [[ -d "$realpath" ]]; then
                eza -1 --icons=always --color=always --group-directories-first "$realpath"
            elif [[ -f "$realpath" ]]; then
                if git rev-parse --is-inside-work-tree &>/dev/null && ! git diff --quiet -- "$realpath" 2>/dev/null; then
                    git diff --color=always -- "$realpath" 2>/dev/null | head -200
                else
                    bat --style=numbers --color=always --line-range :200 "$realpath" 2>/dev/null || cat "$realpath"
                fi
            fi
         else
            local clean_desc="${desc#* -- }"
            local cat_name=""
            local cat_color="\033[1;36m"
            local g_idx="${group#__hide__}"

            # Extraer el color EXACTO que fzf-tab asignó al resultado en la izquierda
            if [[ "$g_idx" =~ ^[0-9]+$ ]]; then
                local -a palette=(
                    $"\033[38;5;75m"
                    $"\033[38;5;221m"
                    $"\033[38;5;176m"
                    $"\033[38;5;81m"
                    $"\033[38;5;204m"
                    $"\033[38;5;114m"
                    $"\033[38;5;209m"
                    $"\033[38;5;141m"
                    $"\033[38;5;43m"
                    $"\033[38;5;211m"
                )
                cat_color="${palette[(( (g_idx - 1) % 10 + 1 ))]}"
            fi

            # Resolver el nombre de la categoría del subcomando en memoria (0 forks)
            case "$word" in
                add|am|archive|bisect|branch|bundle|checkout|cherry-pick|citool|clean|clone|commit|diff|fetch|format-patch|gc|gitk|grep|gui|init|log|maintenance|merge|mv|notes|pull|push|range-diff|rebase|reset|restore|revert|rm|scalar|shortlog|show|sparse-checkout|stash|status|submodule|switch|tag|worktree)
                    cat_name="Main Porcelain" ;;
                config|fast-export|fast-import|filter-branch|mergetool|pack-refs|prune|reflog|remote|repack|replace)
                    cat_name="Ancillary Manipulator" ;;
                annotate|blame|bugreport|count-objects|diagnose|fsck|fsmonitor--daemon|help|instaweb|interpret-trailers|merge-tree|rerere|show-branch|verify-commit|verify-tag|version|whatchanged)
                    cat_name="Ancillary Interrogator" ;;
                apply|checkout-index|commit-graph|commit-tree|hash-object|index-pack|merge-file|merge-index|mktag|mktree|multi-pack-index|pack-objects|prune-packed|read-tree|symbolic-ref|unpack-objects|update-index|update-ref|write-tree)
                    cat_name="Plumbing Manipulator" ;;
                cat-file|check-attr|check-ignore|check-mailmap|check-ref-format|diff-files|diff-index|diff-tree|for-each-ref|for-each-repo|get-tar-commit-id|ls-files|ls-remote|ls-tree|merge-base|name-rev|pack-redundant|rev-list|rev-parse|show-index|show-ref|unpack-file|var|verify-pack)
                    cat_name="Plumbing Interrogator" ;;
                column|fmt-merge-msg|mailinfo|mailsplit|patch-id|stripspace)
                    cat_name="Plumbing Ancillary" ;;
                daemon|fetch-pack|http-backend|send-pack|update-server-info|http-fetch|http-push|receive-pack|shell|upload-archive|upload-pack)
                    cat_name="Plumbing Sync" ;;
                archimport|cvsexportcommit|cvsimport|cvsserver|imap-send|p4|quiltimport|request-pull|send-email|svn)
                    cat_name="Foreign / Interacting" ;;
                *)
                    if [[ -n "$group" && "$group" != *__hide__* && "$group" != \[*\] ]]; then
                        cat_name="$group"
                    fi
                    ;;
            esac

            local cat_badge=""
            if [[ -n "$cat_name" ]]; then
                cat_badge="${cat_color}[${cat_name}]\033[0m"
            fi

            if [[ -n "$clean_desc" && "$clean_desc" != "$word" ]]; then
                if [[ -n "$cat_badge" ]]; then
                    printf "\033[1;36m❯ %s\033[0m  %b\n\n\033[0;37m%s\033[0m\n" "$word" "$cat_badge" "$clean_desc"
                else
                    printf "\033[1;36m❯ %s\033[0m\n\n\033[0;37m%s\033[0m\n" "$word" "$clean_desc"
                fi
            elif [[ -n "$cat_badge" ]]; then
                printf "\033[1;36m❯ %s\033[0m  %b\n" "$word" "$cat_badge"
            fi
         fi'
fi

# ==============================================================================
# 5. ATUIN, PLUGINS WRAPPERS Y PROMPT (ORDEN FINAL ESTRICTO)
# ==============================================================================
if (( $+commands[atuin] )); then
    eval "$(atuin init zsh)"
fi

[[ -f /etc/zsh_command_not_found ]] && source /etc/zsh_command_not_found
[[ -f ~/.zsh/zsh-autosuggestions/zsh-autosuggestions.zsh ]] && source ~/.zsh/zsh-autosuggestions/zsh-autosuggestions.zsh
[[ -f ~/.zsh/fast-syntax-highlighting/fast-syntax-highlighting.plugin.zsh ]] && source ~/.zsh/fast-syntax-highlighting/fast-syntax-highlighting.plugin.zsh

# Starship Prompt (Invocación única)
(( $+commands[starship] )) && eval "$(starship init zsh)"

# Load alias
[[ -f ~/.alias ]] && source ~/.alias
[[ -f ~/.zsh_alias ]] && source ~/.zsh_alias
[[ -f ~/.zsh.alias ]] && source ~/.zsh.alias

# Toggle transparency
function transparent() {
    if [[ -f "$HOME/.config/ghostty/config" ]]; then
        local ghostty_conf="$HOME/.config/ghostty/config"
        if grep -q "^background-opacity = 1" "$ghostty_conf"; then
            sed -i 's/^background-opacity = 2.*/background-opacity = 0.88/' "$ghostty_conf"
            echo "Ghostty transparency enabled (opacity: 0.88)"
        else
            sed -i 's/^background-opacity = .*/background-opacity = 1.0/' "$ghostty_conf"
            echo "Ghostty transparency disabled (opacity: 1.0)"
        fi
    fi
}

alias tt='transparent'

# Clear swap and cache
function cleanup() {
    if [[ "$(uname -s)" == "Linux" ]]; then
        sudo sync
        sudo echo 3 | sudo tee /proc/sys/vm/drop_caches
        sudo swapoff -a
        sudo swapon -a

    elif [[ "$(uname -s)" == "Darwin" ]]; then
        # Swap memory and caches in MacOS does not work exactly the same way as in Linux, you do not have to worry about clearing it manually.
        sudo purge
    fi
}

# Reload env files
function reloadenv() {
    [[ -f /etc/environment ]] && source /etc/environment
    [[ -f ~/.base.zsh ]] && source ~/.base.zsh
    [[ -f ~/.zshrc ]] && source ~/.zshrc
}

function update-go() {
    curl -fsSL https://env.arturonavax.dev/install_golang.sh | bash -s -- $@
}

function update-python() {
    curl -fsSL https://env.arturonavax.dev/install_python.sh | bash -s -- $@
}

# Update
function update() {
    fcwb='\033[1;37m'
    fcr='\033[0m'

    function usage_update(){
        echo -e "$(cat <<EOF
[Updater]: List of update:
  ${fcwb}system ${fcr}/ ${fcwb}s ${fcr}- Upgrades the system.
  ${fcwb}tools ${fcr}/ ${fcwb}t  ${fcr}- Update the tools.
  ${fcwb}all ${fcr}/ ${fcwb}a    ${fcr}- Updates all of the above.
  ${fcwb}help ${fcr}/ ${fcwb}h   ${fcr}- This helpful explanation.${fcr}
EOF
        )"
    }

    for arg in "$@"; do
        case "$arg" in
            tools | t) update_tools=1 ;;
            system | s) update_system=1 ;;
            all | a) update_all=1 ;;
            h | help)
                usage_update

                return 0
                ;;
            *)
                usage_update

                return 1
                ;;
        esac
    done

    [[ "$#" == 0 ]] && update_system=1

    if [[ "$update_all" == 1 ]]; then
        update_tools=1
        update_system=1
    fi

	sudo true

    if [[ "$update_tools" == 1 ]]; then
        if [[ "$(command -v rustup)" != "" ]]; then
            rustup update stable
            rustup check
        fi

        if [[ "$(command -v pnpm)" != "" ]]; then
            pnpm self-update
            pnpm --global update
            [[ -f "$HOME/package.json" ]] && pnpm --dir "$HOME" update
        fi

        if [[ "$(command -v python3)" != "" ]]; then
            python3 -m pip install --upgrade pip
            python3 -m pip install --user --upgrade pipx
            pipx upgrade-all
        fi

        update-go -i
        update-python
    fi

    if [[ "$update_system" == 1 ]]; then
        if [[ "$(uname -s)" == "Linux" ]]; then
            if [[ "$(command -v apt)" != "" ]]; then
                sudo apt update -y
                sudo apt upgrade -y
                sudo apt dist-upgrade -y
                sudo apt full-upgrade -y
                sudo apt autoremove
                sudo apt autoclean
            fi

            if [[ "$(command -v snap)" != "" ]]; then
                sudo snap refresh
            fi

            if [[ "$(command -v dnf)" != "" ]]; then
                sudo dnf update -y
                sudo dnf upgrade -y
                sudo dnf distro-sync -y
                sudo dnf autoremove
            fi

        elif [[ "$(uname -s)" == "Darwin" ]]; then
            sudo softwareupdate -i -a

            if [[ "$(command -v brew)" != "" ]]; then
                brew update
                brew upgrade
                brew cleanup
            fi
        fi
    fi
}

# Load theme mode
[[ -f ~/.theme_mode ]] && export THEME_MODE=$(cat ~/.theme_mode)

# Light theme mode
function lighttheme() {
    if [[ "$(uname -s)" == "Linux" ]]; then
        if [[ "$(command -v gsettings)" != "" ]]; then
            gsettings set org.gnome.desktop.interface color-scheme default
            gsettings set org.gnome.desktop.interface gtk-theme "Yaru-red"
        fi

    elif [[ "$(uname -s)" == "Darwin" ]]; then
        osascript -e 'tell app "System Events" to tell appearance preferences to set dark mode to false' &>/dev/null
    fi

    echo "light" > ~/.theme_mode
    export THEME_MODE=light

    if [[ -f ~/.config/ghostty/auto/theme.ghostty ]]; then
        echo "theme = TokyoNight Day" > ~/.config/ghostty/auto/theme.ghostty
    fi
}

alias li=lighttheme

# Dark mode
function darktheme() {
    if [[ "$(uname -s)" == "Linux" ]]; then
        if [[ "$(command -v gsettings)" != "" ]]; then
            gsettings set org.gnome.desktop.interface color-scheme prefer-dark
            gsettings set org.gnome.desktop.interface gtk-theme "Yaru-red-dark"
        fi

    elif [[ "$(uname -s)" == "Darwin" ]]; then
        osascript -e 'tell app "System Events" to tell appearance preferences to set dark mode to true' &>/dev/null
    fi

    echo "dark" > ~/.theme_mode
    export THEME_MODE=dark

    if [[ -f ~/.config/ghostty/auto/theme.ghostty ]]; then
        echo "theme = TokyoNight Night" > ~/.config/ghostty/auto/theme.ghostty
    fi
}

alias da=darktheme

# Copy current path
function cpath() {
    if [[ "$(uname -s)" == "Linux" && "$(command -v xclip)" != "" ]]; then
        pwd | tr -d '\n' | xclip -selection clipboard

    elif [[ "$(uname -s)" == "Darwin" ]]; then
        pwd | tr -d '\n' | pbcopy
    fi
}

# Flush the DNS Cache
function flushdns() {
    if [[ "$(uname -s)" == "Linux" ]]; then
        sudo systemd-resolve --flush-caches

    elif [[ "$(uname -s)" == "Darwin" ]]; then
        sudo dscacheutil -flushcache
        sudo killall -HUP mDNSResponder
        sudo killall mDNSResponderHelper
    fi
}

# Get Public IP
alias publicip='curl --silent ipinfo.io/ip'

# Get Public IP Geolocation
function geoip() {
    fgcolor_reset='\033[0m'
    fgcolor_white_bold='\033[1;37m'
    fgcolor_blue_bold='\033[1;34m'
    fgcolor_green_bold='\033[1;32m'

    if [[ "$#" == 0 ]]; then
        echo "${fgcolor_white_bold}My GeoIP:${fgcolor_reset}"

        curl --silent ipinfo.io/"$(curl --silent ipinfo.io/ip)" | json_pp

    else
        for arg in "$@"; do
            echo "${fgcolor_white_bold}GeoIP ${fgcolor_green_bold}$arg$fgcolor_white_bold:${fgcolor_reset}"

            curl --silent "ipinfo.io/$arg" | json_pp

            echo "${fgcolor_blue_bold}---${fgcolor_reset}"
        done
    fi
}

# Create a new project in Go
function gonew() {
    [[ "$1" == "" ]] && 1="sandbox"

    mkdir "$1" || return 1

    cd "$1" || return

    go mod init "$1"

    echo -e 'package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println("Hello world!")\n}' >main.go
}

# Clear
alias clear="printf '\33c\e[3J'"

# Print PATH
alias path='echo "${PATH//:/\n}" | sort'

# Set tools
## Ripgrep (Preservar grep POSIX nativo; atajo ergonomico para busqueda oculta)
[[ "$(command -v rg)" != "" ]] && alias rgh='rg --hidden'

## Fd (Normalizar nombre de paquete en Debian/Ubuntu sin romper GNU find)
if [[ "$(command -v fdfind)" != "" && "$(command -v fd)" == "" ]]; then
    alias fd='fdfind'
fi
[[ "$(command -v fd)" != "" || "$(command -v fdfind)" != "" ]] && alias fdh='fd --hidden'

## Bat (Normalizar nombre en Debian/Ubuntu sin secuestrar cat o less nativos)
if [[ "$(command -v batcat)" != "" && "$(command -v bat)" == "" ]]; then
    alias bat='batcat'
fi
[[ "$(command -v bat)" != "" || "$(command -v batcat)" != "" ]] && alias preview='bat'

## Difftastic (Diff estructural AST basado en Tree-sitter)
[[ "$(command -v difft)" != "" ]] && alias dft='difft'

## Eza (Atajos modernos con iconos y clasificacion manteniendo compatibilidad)
if [[ "$(command -v eza)" != "" ]]; then
    alias l='eza --icons --classify=always'
    alias ll='eza --icons --classify=always -lh'
    alias la='eza --icons --classify=always -lha'
    alias lt='eza --icons --classify=always --tree'
    alias llt='eza --icons --classify=always --tree -lh'
    alias ls='eza --icons --classify=always'
else
    alias ls='ls --color=auto'
    alias ll='ls --color=auto -l'
    alias la='ls --color=auto -la'
fi

## zoxide
if [[ "$(command -v zoxide)" != "" ]]; then
    eval "$(zoxide init zsh)"
fi

## direnv
if [[ "$(command -v direnv)" != "" ]]; then
    eval "$(direnv hook zsh)"
fi

## GPG TTY
export GPG_TTY=$(tty 2>/dev/null || echo "")

## Cursor IDE alias (AppImage or system binary)
if [[ "$(command -v cursor)" == "" && -d "$HOME/Applications" ]]; then
    cursor_appimage=$(find "$HOME/Applications" -maxdepth 1 -name "Cursor-*.AppImage" 2>/dev/null | head -n 1)
    [[ -n "$cursor_appimage" ]] && alias cursor="$cursor_appimage --no-sandbox"
fi

## Notificaciones automáticas de comandos largos (> 4s) en segundo plano (Ghostty, Herdr, Xfce, tmux)
zmodload zsh/datetime 2>/dev/null || :
autoload -Uz add-zsh-hook 2>/dev/null || :

export NOTIFY_THRESHOLD=4

__is_interactive_cmd() {
    local raw_cmd="$1"
    local first_cmd="${raw_cmd%% *}"
    first_cmd="${first_cmd##*/}"

    # Desenvolver invocaciones a través de sudo, doas, env o nohup
    if [[ "$first_cmd" == "sudo" || "$first_cmd" == "doas" || "$first_cmd" == "env" || "$first_cmd" == "nohup" ]]; then
        local rest="${raw_cmd#* }"
        first_cmd="${rest%% *}"
        first_cmd="${first_cmd##*/}"
    fi

    # Editores, paginadores, visores interactivos y multiplexores
    case "$first_cmd" in
        nvim|vim|vi|nano|emacs|helix|hx|pico|joe|micro|kak) return 0 ;;
        less|more|most|bat|batcat|man|info) return 0 ;;
        top|htop|btop|glances|nvtop|gotop|iftop|iotop) return 0 ;;
        lazygit|tig|gitui) return 0 ;;
        ssh|mosh|telnet|ftp|sftp) return 0 ;;
        ranger|vifm|yazi|nnn|mc) return 0 ;;
        herdr|tmux|screen|zellij) return 0 ;;
        fzf|peco|fzy) return 0 ;;
        agy|claude|chatgpt) return 0 ;;
    esac

    # Subcomandos interactivos de git
    if [[ "$first_cmd" == "git" ]]; then
        local sub="${raw_cmd#*git }"
        sub="${sub%% *}"
        case "$sub" in
            commit|log|diff|show|rebase) return 0 ;;
        esac
    fi

    return 1
}

__notify_preexec() {
    __cmd_start=$EPOCHSECONDS
    __cmd_name="$1"
}

__notify_precmd() {
    local exit_code=$?

    if [[ -n "$__cmd_start" ]]; then
        local elapsed=$(( EPOCHSECONDS - __cmd_start ))
        local cmd="$__cmd_name"

        unset __cmd_start __cmd_name

        # Omitir editores y herramientas interactivas
        if __is_interactive_cmd "$cmd"; then
            return
        fi

        if (( elapsed >= ${NOTIFY_THRESHOLD:-4} )); then
            local should_notify=0
            local is_focused=1

            # 1. Comprobar foco del pane en Herdr si se está dentro de herdr
            if [[ -n "$HERDR_PANE_ID" ]] && command -v herdr &>/dev/null; then
                if ! herdr pane get "$HERDR_PANE_ID" 2>/dev/null | grep -q '"focused":true'; then
                    is_focused=0
                fi
            fi

            # 2. Comprobar foco del pane en Tmux si se está dentro de tmux
            if [[ -n "$TMUX" ]] && command -v tmux &>/dev/null; then
                local pane_target="${TMUX_PANE:-}"
                local tmux_focused
                if [[ -n "$pane_target" ]]; then
                    tmux_focused=$(tmux display-message -p -t "$pane_target" '#{&&:#{session_attached},#{&&:#{m:*focused*,#{client_flags}},#{&&:#{pane_active},#{window_active}}}}' 2>/dev/null)
                else
                    tmux_focused=$(tmux display-message -p '#{&&:#{session_attached},#{&&:#{m:*focused*,#{client_flags}},#{&&:#{pane_active},#{window_active}}}}' 2>/dev/null)
                fi
                [[ "$tmux_focused" != "1" ]] && is_focused=0
            fi

            # 3. Comprobar foco de ventana a nivel de escritorio X11
            if command -v xdotool &>/dev/null; then
                local active_win
                active_win=$(xdotool getactivewindow 2>/dev/null)
                if [[ -n "$active_win" ]]; then
                    if [[ -n "$WINDOWID" ]]; then
                        [[ "$active_win" != "$WINDOWID" ]] && is_focused=0
                    else
                        local active_class
                        active_class=$(xprop -id "$active_win" WM_CLASS 2>/dev/null)
                        [[ "$active_class" != *"ghostty"* && "$active_class" != *"terminal"* ]] && is_focused=0
                    fi
                fi
            fi

            # Notificar si la ventana o el panel están en segundo plano (o si se fuerza siempre)
            if [[ "${NOTIFY_UNFOCUSED_ONLY:-1}" == "0" ]] || (( ! is_focused )); then
                should_notify=1
            fi

            if (( should_notify )) && command -v notify-send &>/dev/null; then
                local title="Comando finalizado (${elapsed}s)"
                local urgency="normal"

                if (( exit_code != 0 )); then
                    title="Comando fallido (código $exit_code) (${elapsed}s)"
                    urgency="critical"
                fi

                # Truncar comandos extensos para evitar desbordes visuales
                if (( ${#cmd} > 80 )); then
                    cmd="${cmd[1,77]}..."
                fi

                notify-send -u "$urgency" -i utilities-terminal -a "Terminal" "$title" "$cmd" >/dev/null 2>&1 || true
            fi
        fi
    fi
}

add-zsh-hook preexec __notify_preexec 2>/dev/null || :
add-zsh-hook precmd __notify_precmd 2>/dev/null || :

## Compile Zsh startup files to word-code (.zwc) for ultra-fast startup
zcompile-all() {
    local file
    for file in "${ZDOTDIR:-$HOME}"/.zshrc "${ZDOTDIR:-$HOME}"/.base.zsh "${ZDOTDIR:-$HOME}"/.tools.sh "${ZDOTDIR:-$HOME}"/.lscolors.sh; do
        if [[ -f "$file" ]]; then
            if [[ ! -f "${file}.zwc" || "$file" -nt "${file}.zwc" ]]; then
                zcompile "$file" && echo "Compiled: ${file}.zwc"
            fi
        fi
    done
}


## Bun completions
if [[ -d "$HOME/.bun" && -s "$HOME/.bun/_bun" ]]; then
    source "$HOME/.bun/_bun"
fi
:
