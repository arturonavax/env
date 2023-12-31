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

editor="lvim"

function usage() {
	echo -e "$(
		cat <<EOF
[Sync Config]: List of Sync:
  ${fcwb}terminal ${fcr}/ ${fcwb}t ${fcr}- Synchronize the Terminal settings.
  ${fcwb}editor ${fcr}/ ${fcwb}e   ${fcr}- Synchronize Editor ($editor) settings.
  ${fcwb}osconfig ${fcr}/ ${fcwb}o ${fcr}- Configure the operating system with personal preferences.
  ${fcwb}plugins ${fcr}/ ${fcwb}p  ${fcr}- Synchronize Editor ($editor) plugins.
  ${fcwb}vscode ${fcr}/ ${fcwb}v   ${fcr}- Synchronize Visual Studio Code settings.
  ${fcwb}devtools ${fcr}/ ${fcwb}d ${fcr}- Synchronizes the configuration of development tools.
  ${fcwb}all ${fcr}/ ${fcwb}a      ${fcr}- Synchronizes the configuration of all of the above.
  ${fcwb}help ${fcr}/ ${fcwb}h     ${fcr}- This helpful explanaiton.${fcr}
EOF
	)"
}

for arg in "$@"; do
	case "$arg" in
	terminal | t) sync_terminal=1 ;;
	editor | e) sync_editor=1 ;;
	plugins | p) sync_plugins=1 ;;
	osconfig | o) sync_osconfig=1 ;;
	vscode | v) sync_vscode=1 ;;
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
	sync_vscode=1
	sync_devtools=1
fi

# Sync Terminal files
if [[ "$sync_terminal" == 1 ]]; then
	mkdir -p ~/.config/alacritty/

	cp ./files/alacritty/alacritty.toml ~/.config/alacritty/.
	cp ./files/alacritty/dark_theme.toml ~/.config/alacritty/.
	cp ./files/alacritty/light_theme.toml ~/.config/alacritty/.

	cp ./files/tmux/.tmux.conf ~/.

	cp ./files/starship/starship.toml ~/.config/.

	cp ./files/zsh/.lscolors.sh ~/.
	cp ./files/zsh/.base.zsh ~/.
	cp ./files/zsh/.tools.sh ~/.

	echo -e "${fcgreenb}Synchronized Terminal configuration files!${fcr}"
fi

# Sync Editor files
if [[ "$sync_editor" == 1 ]]; then
	mkdir -p ~/.local/share/lunarvim/lvim/
	mkdir -p ~/.config/lvim/lsp-settings/
	mkdir -p ~/.config/lvim/snippets/
	mkdir -p ~/.config/lvim/lua/user/

	touch ~/.config/lvim/.root

	cp ./files/lvim/default.json ~/.local/share/lunarvim/lvim/snapshots/.

	cp ./files/lvim/config.lua ~/.config/lvim/.
	cp ./files/lvim/lua/user/* ~/.config/lvim/lua/user/.
	cp ./files/snippets/* ~/.config/lvim/snippets/.
	cp ./files/lsp-settings/* ~/.config/lvim/lsp-settings/.

	echo -e "${fcgreenb}Synchronized Editor ($editor) configuration files!${fcr}"
fi

# Sync VSCode files
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

	echo -e "${fcgreenb}Synchronized Code configuration files!${fcr}"
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

# Sync Editor plugins
if [[ "$sync_plugins" == 1 ]]; then
	echo "Synchronizing Editor ($editor) plugins..."

	cp ./files/lvim/default.json ~/.local/share/lunarvim/lvim/snapshots/.

	if [[ "$(command -v lvim)" != "" ]]; then
		lvim --headless +LvimSyncCorePlugins +qa 2>/dev/null
	fi

	echo -e "${fcgreenb}Synchronized Editor ($editor) plugins!${fcr}"
fi

# Sync OS config
if [[ "$sync_osconfig" == 1 ]]; then
	echo "Synchronizing OS config..."

	if [[ "$(uname -s)" == "Linux" ]]; then
		./src/remotes/linux_osconfig.sh

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		./src/remotes/macos_osconfig.sh
	fi

	cp ./files/amethyst/.amethyst.yml ~/.

	echo -e "${fcgreenb}Synchronized OS config!${fcr}"
fi
