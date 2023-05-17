#!/bin/zsh
bindkey -e

[[ "$(command -v lvim)" != "" ]] && export EDITOR=lvim

# load zsh-completions
autoload -Uz compinit && compinit

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
    HISTSIZE=1000000\
    SAVEHIST=1000000

setopt HIST_FIND_NO_DUPS SHARE_HISTORY

## utf8
export \
    LC_ALL=en_US.UTF-8 \
    LANG=en_US.UTF-8 \
    LANGUAGE=en_US.UTF-8

# fzf
[[ -f ~/.fzf.zsh ]] && source ~/.fzf.zsh

## zsh plugins
[[ -f /etc/zsh_command_not_found ]] && source /etc/zsh_command_not_found
[[ -f ~/.zsh/fzf-tab/fzf-tab.plugin.zsh ]] && source ~/.zsh/fzf-tab/fzf-tab.plugin.zsh
[[ -f ~/.zsh/zsh-autosuggestions/zsh-autosuggestions.zsh ]] && source ~/.zsh/zsh-autosuggestions/zsh-autosuggestions.zsh
[[ -f ~/.zsh/fast-syntax-highlighting/fast-syntax-highlighting.plugin.zsh ]] && source ~/.zsh/fast-syntax-highlighting/fast-syntax-highlighting.plugin.zsh

export \
    fpath=(~/.zsh/zsh-completions/src $fpath) \
    FZF_DEFAULT_COMMAND="rg --color=always --files --no-ignore --hidden --follow" \
    FZF_DEFAULT_OPTS="--reverse --prompt '❯ ' --pointer ''" \
    FZF_CTRL_R_OPTS="--no-info --color prompt:italic --prompt '  History ❯ '"

zstyle ":completion:*" list-colors ${(s.:.)LS_COLORS}
zstyle ":fzf-tab:*" fzf-command ftb-tmux-popup
zstyle ":fzf-tab:*" fzf-flags "--no-info" "--prompt=❯ " "--pointer="

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
    origen_file="$HOME/.config/alacritty/alacritty.yml"

    content_new="$(awk '
	{
		if (/[^#]opacity:/) {
			gsub ("opacity:", "#opacity:");
		} else if (/#opacity:/) {
		gsub ("#opacity:", "opacity:");
	}

	print
}
    ' "$origen_file")"

    echo "$content_new" >"$origen_file"
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
        echo "!!! Under construction for MacOS"
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
            system | s) update_system=1 ;;
            tools | t) update_tools=1 ;;
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
        update_system=1
        update_tools=1
    fi

    if [[ "$update_system" == 1 ]]; then
        if [[ "$(uname -s)" == "Linux" ]]; then
            sudo apt -y update
            sudo apt -y upgrade
            sudo apt -y dist-upgrade
            sudo apt -y full-upgrade
            sudo apt autoremove
            sudo apt autoclean

            if [[ "$(command -v snap)" != "" ]]; then
                sudo snap refresh
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

    if [[ "$update_tools" == 1 ]]; then
        if [[ "$(command -v rustup)" != "" ]]; then
            rustup update stable
            rustup check
        fi

        if [[ "$(command -v pnpm)" != "" ]]; then
            pnpm add -g @pnpm/exe
            pnpm --global update
            cd "$HOME"
            pnpm update
            cd -
        fi

        if [[ "$(command -v python3)" != "" ]]; then
            python3 -m pip install --upgrade pip
            python3 -m pip install --user --upgrade pipx
            pipx upgrade-all
        fi

        update-go -i
    fi
}

# Load theme mode
[[ -f ~/.theme_mode ]] && export THEME_MODE=$(cat ~/.theme_mode)

# Light theme mode
function lighttheme() {
    if [[ "$(uname -s)" == "Linux" ]]; then
        gsettings set org.gnome.desktop.interface color-scheme default
        gsettings set org.gnome.desktop.interface gtk-theme "Yaru-red"

    elif [[ "$(uname -s)" == "Darwin" ]]; then
        osascript -e 'tell app "System Events" to tell appearance preferences to set dark mode to false' &>/dev/null
    fi

    echo "light" > ~/.theme_mode
    export THEME_MODE=light

    tmux setenv THEME_MODE light &>/dev/null

    [[ -f ~/.tmux/plugins/tmux-theme/tmux-theme.tmux ]] && bash ~/.tmux/plugins/tmux-theme/tmux-theme.tmux &>/dev/null
    [[ -f ~/.tmux/plugins/tmux-battery/battery.tmux ]] && bash ~/.tmux/plugins/tmux-battery/battery.tmux &>/dev/null

    [[ -f ~/.config/alacritty/light_theme.yml ]] && cp ~/.config/alacritty/light_theme.yml ~/.config/alacritty/alacritty.yml
}

alias li=lighttheme

# Dark mode
function darktheme() {
    if [[ "$(uname -s)" == "Linux" ]]; then
        gsettings set org.gnome.desktop.interface color-scheme prefer-dark
        gsettings set org.gnome.desktop.interface gtk-theme "Yaru-red-dark"

    elif [[ "$(uname -s)" == "Darwin" ]]; then
        osascript -e 'tell app "System Events" to tell appearance preferences to set dark mode to true' &>/dev/null
    fi

    echo "dark" > ~/.theme_mode
    export THEME_MODE=dark
    tmux setenv THEME_MODE dark &>/dev/null

    [[ -f ~/.tmux/plugins/tmux-theme/tmux-theme.tmux ]] && bash ~/.tmux/plugins/tmux-theme/tmux-theme.tmux &>/dev/null
    [[ -f ~/.tmux/plugins/tmux-battery/battery.tmux ]] && bash ~/.tmux/plugins/tmux-battery/battery.tmux &>/dev/null

    [[ -f ~/.config/alacritty/dark_theme.yml ]] && cp ~/.config/alacritty/dark_theme.yml ~/.config/alacritty/alacritty.yml
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
        sudo killall -HUP mDNSResponder
        sudo killall mDNSResponderHelper
        sudo dscacheutil -flushcache
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
    mkdir "$1"

    cd "$1" || return

    go mod init "$1"

    echo -e 'package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println("Hello world!")\n}' >main.go
}

# Clear
alias clear="printf '\33c\e[3J'"

# Print PATH
alias path='echo "${PATH//:/\n}" | sort'

# Set tools
## Ripgrep
[[ "$(command -v rg)" != "" ]] && alias grep='rg --hidden'

## Fd
if fd -q &>/dev/null; then
    alias fdfind='fd'
    alias find='fdfind --hidden'
fi

## Exa
if [[ "$(command -v exa)" != "" ]]; then
    alias ls='exa --icons --classify'
    alias ll='exa --icons --classify -lh'
    alias llt='exa --icons --classify --tree'

else
    alias ls='ls --color --classify'
    alias ll='ls --color --classify -l'
fi

## Bat
if [[ "$(command -v batcat)" != "" || "$(command -v bat)" != "" ]]; then
    [[ "$(command -v bat)" != "" ]] && alias batcat='bat'
    alias cat='batcat -P -p'
    alias less='batcat'
fi

## zoxide
if [[ "$(command -v zoxide)" != "" ]]; then
    eval "$(zoxide init zsh)"
fi

# PATH
## Basics
[[ ":$PATH:" != *":/usr/local/bin:"* ]] && export PATH="$PATH:/usr/local/bin"
[[ ":$PATH:" != *":$HOME/.local/bin:"* ]] && export PATH="$HOME/.local/bin:$PATH"

## MacOS
if [[ "$(uname -s)" == "Darwin" ]]; then
    brewbin="/usr/local/bin/brew"
    [[ ! -f "$brewbin" ]] && brewbin="/opt/homebrew/bin/brew" # For Apple Silicon
    [[ -f "$brewbin" ]] && eval "$("$brewbin" shellenv)"
fi
:
