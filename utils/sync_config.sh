#!/bin/bash
# This script sync the configuration files from the repository into your file system.
# This script is made to run from the repository root.
#
# Run: ./utils/sync_config.sh all
while [[ ! -d ./.git/ && ! -d ./files/ && ! -d ./src/ ]]; do
	if [[ "$PWD" == "/" ]]; then
		echo "Repository root not found"

		exit 1
	fi

	if ! cd ..; then
		echo "An error occurred while doing 'cd ..'"

		exit 1
	fi
done

fcr='\033[0m'
fcwb='\033[1;37m'
fcgreenb='\033[1;32m'

editor="nvim"

function usage() {
	echo -e "$(
		cat <<EOF
[Sync Config]: List of Sync:
  ${fcwb}terminal ${fcr}/ ${fcwb}t       ${fcr}- Synchronize the Terminal settings (Ghostty, tmux, starship, zsh).
  ${fcwb}editor ${fcr}/ ${fcwb}e         ${fcr}- Synchronize Editor ($editor - LazyVim) settings.
  ${fcwb}osconfig ${fcr}/ ${fcwb}o       ${fcr}- Configure the operating system with personal preferences.
  ${fcwb}plugins ${fcr}/ ${fcwb}p        ${fcr}- Restore/synchronize Editor ($editor - LazyVim) plugins from lockfile.
  ${fcwb}plugins-update ${fcr}/ ${fcwb}up ${fcr}- Update Editor ($editor - LazyVim) plugins and lockfile.
  ${fcwb}vscode ${fcr}/ ${fcwb}v         ${fcr}- Synchronize Visual Studio Code settings (explicit argument only).
  ${fcwb}cursor ${fcr}/ ${fcwb}c         ${fcr}- Synchronize Cursor IDE settings (explicit argument only).
  ${fcwb}devtools ${fcr}/ ${fcwb}d       ${fcr}- Synchronizes the configuration of development tools.
  ${fcwb}all ${fcr}/ ${fcwb}a            ${fcr}- Synchronizes all base configurations (excludes GUI IDEs).
  ${fcwb}help ${fcr}/ ${fcwb}h           ${fcr}- This helpful explanaiton.${fcr}
EOF
	)"
}

for arg in "$@"; do
	case "$arg" in
	terminal | t) sync_terminal=1 ;;
	editor | e) sync_editor=1 ;;
	plugins | p) sync_plugins=1 ;;
	plugins-update | up) update_plugins=1 ;;
	osconfig | o) sync_osconfig=1 ;;
	vscode | v) sync_vscode=1 ;;
	cursor | c) sync_cursor=1 ;;
	devtools | d) sync_devtools=1 ;;
	all | a) sync_all=1 ;;
	h | help)
		usage

		exit 0
		;;
	*)
		usage

		exit 1
		;;
	esac
done

[[ "$#" == 0 ]] && sync_all=1

if [[ "$sync_all" == 1 ]]; then
	sync_terminal=1
	sync_editor=1
	sync_plugins=1
	sync_osconfig=1
	sync_devtools=1
	# Note: VSCode and Cursor are NOT synchronized automatically by 'all' or by default.
	# They must be specified explicitly via 'vscode'/'v' or 'cursor'/'c' argument.
fi

# Sync Terminal files
if [[ "$sync_terminal" == 1 ]]; then
	mkdir -p ~/.config/ghostty/auto/

	cp ./files/ghostty/config ~/.config/ghostty/.
	[[ -f ./files/ghostty/auto/theme.ghostty ]] && cp ./files/ghostty/auto/theme.ghostty ~/.config/ghostty/auto/.

	cp ./files/tmux/.tmux.conf ~/.

	cp ./files/starship/starship.toml ~/.config/.

	cp ./files/zsh/.lscolors.sh ~/.
	cp ./files/zsh/.base.zsh ~/.
	cp ./files/zsh/.tools.sh ~/.

	echo -e "${fcgreenb}Synchronized Terminal configuration files!${fcr}"
fi

# Sync Editor files
if [[ "$sync_editor" == 1 ]]; then
	mkdir -p ~/.config/nvim/lua/config/ ~/.config/nvim/lua/plugins/

	cp ./files/nvim/init.lua ~/.config/nvim/.
	[[ -f ./files/nvim/lazyvim.json ]] && cp ./files/nvim/lazyvim.json ~/.config/nvim/.
	[[ -f ./files/nvim/stylua.toml ]] && cp ./files/nvim/stylua.toml ~/.config/nvim/.
	[[ -f ./files/nvim/.gitignore ]] && cp ./files/nvim/.gitignore ~/.config/nvim/.
	[[ -f ./files/nvim/.neoconf.json ]] && cp ./files/nvim/.neoconf.json ~/.config/nvim/.
	[[ -f ./files/nvim/lazy-lock.json ]] && cp ./files/nvim/lazy-lock.json ~/.config/nvim/.

	cp ./files/nvim/lua/config/*.lua ~/.config/nvim/lua/config/. 2>/dev/null || :
	cp ./files/nvim/lua/plugins/*.lua ~/.config/nvim/lua/plugins/. 2>/dev/null || :

	echo -e "${fcgreenb}Synchronized Editor ($editor - LazyVim) configuration files!${fcr}"
fi

# Sync VSCode files (explicit argument only)
if [[ "$sync_vscode" == 1 ]]; then
	if [[ "$(uname -s)" == "Linux" ]]; then
		mkdir -p ~/.config/Code/User/
		cp ./files/vscode/settings.json ~/.config/Code/User/.
		cp ./files/vscode/keybindings.json ~/.config/Code/User/.

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		mkdir -p "$HOME/Library/Application Support/Code/User/"
		cp ./files/vscode/settings.json "$HOME/Library/Application Support/Code/User/."
		cp ./files/vscode/keybindings.json "$HOME/Library/Application Support/Code/User/."
	fi

	echo -e "${fcgreenb}Synchronized Visual Studio Code configuration files!${fcr}"
fi

# Sync Cursor files (explicit argument only)
if [[ "$sync_cursor" == 1 ]]; then
	if [[ "$(uname -s)" == "Linux" ]]; then
		mkdir -p ~/.config/Cursor/User/
		cp ./files/cursor/settings.json ~/.config/Cursor/User/.
		cp ./files/cursor/keybindings.json ~/.config/Cursor/User/.

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		mkdir -p "$HOME/Library/Application Support/Cursor/User/"
		cp ./files/cursor/settings.json "$HOME/Library/Application Support/Cursor/User/."
		cp ./files/cursor/keybindings.json "$HOME/Library/Application Support/Cursor/User/."
	fi

	echo -e "${fcgreenb}Synchronized Cursor IDE configuration files!${fcr}"
fi

# Sync Devtools files
if [[ "$sync_devtools" == 1 ]]; then
	cp ./files/golangci-lint/.golangci.yml ~/.
	cp ./files/eslint/.eslintrc.json ~/.
	cp ./files/prettier/.prettierrc.json ~/.
	cp ./files/stylelint/.stylelintrc.json ~/.
	cp ./files/sql-formatter/.sql-formatter.json ~/.

	echo -e "${fcgreenb}Synchronized Devtools configuration files!${fcr}"
fi

# Sync Editor plugins (restore from lockfile)
if [[ "$sync_plugins" == 1 ]]; then
	echo "Restoring/Synchronizing Editor ($editor - LazyVim) plugins from lockfile..."

	if [[ "$(command -v nvim)" != "" ]]; then
		nvim --headless "+Lazy! restore" +qa 2>/dev/null || :
	fi

	echo -e "${fcgreenb}Synchronized Editor ($editor - LazyVim) plugins!${fcr}"
fi

# Update Editor plugins and refresh lockfile in repository
if [[ "$update_plugins" == 1 ]]; then
	echo "Updating Editor ($editor - LazyVim) plugins..."

	if [[ "$(command -v nvim)" != "" ]]; then
		nvim --headless "+Lazy! update" +qa 2>/dev/null || :
		if [[ -f ~/.config/nvim/lazy-lock.json ]]; then
			cp ~/.config/nvim/lazy-lock.json ./files/nvim/.
			echo -e "${fcgreenb}Updated ./files/nvim/lazy-lock.json with new plugin versions!${fcr}"
		fi
	fi

	echo -e "${fcgreenb}Updated Editor ($editor - LazyVim) plugins!${fcr}"
fi

# Sync OS config
if [[ "$sync_osconfig" == 1 ]]; then
	echo "Synchronizing OS config..."

	if [[ "$(uname -s)" == "Linux" ]]; then
		mkdir -p ~/.config/rcmd/ ~/.local/lib/rcmd/backends/ ~/.local/bin/
		if [[ -f ./files/rcmd/rcmd.conf && ! -f ~/.config/rcmd/rcmd.conf ]]; then
			cp ./files/rcmd/rcmd.conf ~/.config/rcmd/.
		fi
		cp ./files/rcmd/backends/*.sh ~/.local/lib/rcmd/backends/. 2>/dev/null || :
		cp ./files/rcmd/rcmd ~/.local/bin/rcmd 2>/dev/null || :
		chmod +x ~/.local/bin/rcmd ~/.local/lib/rcmd/backends/*.sh 2>/dev/null || :
		./src/remotes/linux_osconfig.sh

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		./src/remotes/macos_osconfig.sh
		cp ./files/amethyst/.amethyst.yml ~/.
		mkdir -p ~/.config/karabiner/
		cp ./files/karabiner/karabiner.json ~/.config/karabiner/.
	fi

	echo -e "${fcgreenb}Synchronized OS config!${fcr}"
fi
