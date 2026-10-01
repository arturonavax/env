#!/bin/zsh
[[ "$0" != *"zsh"* ]] && return

bindkey -e

# PATH Basics
[[ ":$PATH:" != *":/usr/local/bin:"* ]] && export PATH="$PATH:/usr/local/bin"
[[ ":$PATH:" != *":$HOME/.local/bin:"* ]] && export PATH="$HOME/.local/bin:$PATH"
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

# load zsh-completions with cached compdump
autoload -Uz compinit
if [[ -n "${ZDOTDIR:-$HOME}/.zcompdump(#qN.mh+24)" ]]; then
    compinit
    zcompile -R "${ZDOTDIR:-$HOME}/.zcompdump" 2>/dev/null || :
else
    compinit -C
fi

# enable zsh comments
setopt interactivecomments

# zsh ask for confirmation with !!
setopt histverify

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

# Integración nativa de fzf con Zsh (atajos Ctrl+R, Ctrl+T, Alt+C y completion)
if [[ "$(command -v fzf)" != "" ]]; then
    eval "$(fzf --zsh)"

    # Usar fd en lugar de find clásico (ignora .git, respeta .gitignore y sigue symlinks)
    export FZF_DEFAULT_COMMAND='fd --type f --strip-cwd-prefix --hidden --follow --exclude .git'
    export FZF_CTRL_T_COMMAND="$FZF_DEFAULT_COMMAND"
    export FZF_ALT_C_COMMAND='fd --type d --strip-cwd-prefix --hidden --follow --exclude .git'

    # Previews asíncronos y ligeros (ocultos por defecto con :hidden para evitar overhead de I/O, toggle con Ctrl+/)
    export FZF_DEFAULT_OPTS="--height 40% --layout=reverse --border --inline-info --preview 'if [ -d {} ]; then eza --tree --level=2 --color=always {} 2>/dev/null | head -200; else bat --style=numbers --color=always --line-range :300 {} 2>/dev/null || cat {}; fi' --preview-window right:60%:hidden:wrap --bind 'ctrl-/:toggle-preview'"
    export FZF_CTRL_T_OPTS="--preview 'bat --style=numbers --color=always --line-range :300 {} 2>/dev/null || cat {}' --preview-window right:60%:hidden:wrap --bind 'ctrl-/:toggle-preview'"
    export FZF_ALT_C_OPTS="--preview 'eza --tree --level=2 --icons --color=always {} 2>/dev/null' --preview-window right:60%:hidden:wrap --bind 'ctrl-/:toggle-preview'"
fi

## Atuin (SQLite-backed ultra-fast shell history replacing Ctrl-R)
if [[ "$(command -v atuin)" != "" ]]; then
    eval "$(atuin init zsh)"
fi

## zsh plugins
[[ -f /etc/zsh_command_not_found ]] && source /etc/zsh_command_not_found
[[ -f ~/.zsh/zsh-autosuggestions/zsh-autosuggestions.zsh ]] && source ~/.zsh/zsh-autosuggestions/zsh-autosuggestions.zsh
[[ -f ~/.zsh/fast-syntax-highlighting/fast-syntax-highlighting.plugin.zsh ]] && source ~/.zsh/fast-syntax-highlighting/fast-syntax-highlighting.plugin.zsh

export fpath=(~/.zsh/zsh-completions/src $fpath)
zstyle ":completion:*" list-colors ${(s.:.)LS_COLORS}

# Better SSH/Rsync/SCP Autocomplete
zstyle ':completion:*:(scp|rsync):*' tag-order ' hosts:-ipaddr:ip\ address hosts:-host:host files'
zstyle ':completion:*:(ssh|scp|rsync):*:hosts-host' ignored-patterns '*(.|:)*' loopback ip6-loopback localhost ip6-localhost broadcasthost
zstyle ':completion:*:(ssh|scp|rsync):*:hosts-ipaddr' ignored-patterns '^(<->.<->.<->.<->|(|::)([[:xdigit:].]##:(#c,2))##(|%*))' '127.0.0.<->' '255.255.255.255' '::1' 'fe80::*'

# Allow for autocomplete to be case insensitive
zstyle ':completion:*' matcher-list '' 'm:{[:lower:][:upper:]-_}={[:upper:][:lower:]_-}' \
    '+l:|?=** r:|?=**'

# prompt starship
[[ "$(command -v starship)" != "" ]] && eval "$(starship init zsh)"

# Load alias
[[ -f ~/.alias ]] && source ~/.alias
[[ -f ~/.zsh_alias ]] && source ~/.zsh_alias
[[ -f ~/.zsh.alias ]] && source ~/.zsh.alias

# Toggle transparency
function transparent() {
    if [[ -f "$HOME/.config/ghostty/config" ]]; then
        local ghostty_conf="$HOME/.config/ghostty/config"
        if grep -q "^background-opacity = 1" "$ghostty_conf"; then
            sed -i 's/^background-opacity = 1.*/background-opacity = 0.88/' "$ghostty_conf"
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
    alias l='eza --icons --classify'
    alias ll='eza --icons --classify -lh'
    alias la='eza --icons --classify -lha'
    alias lt='eza --icons --classify --tree'
    alias llt='eza --icons --classify --tree -lh'
    alias ls='eza --icons --classify'
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
