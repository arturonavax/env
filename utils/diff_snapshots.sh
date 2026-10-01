#!/bin/bash
# This script inspects differences between the repo configuration files and the local system.
# Run: ./utils/diff_snapshots.sh
# shellcheck disable=SC2154
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

source ./src/remotes/_vars_colors.sh

# set viewer / diff highlighter
if [[ "$(command -v delta)" != "" ]]; then
	viewer="delta --paging=never"

elif [[ "$(command -v batcat)" != "" ]]; then
	viewer="batcat"

elif [[ "$(command -v bat)" != "" ]]; then
	viewer="bat"

elif [[ "$(command -v less)" != "" ]]; then
	viewer="less"

else
	viewer="cat"
fi

echo -e "${fgcolor_white_bold}[Diff Config]: Checking differences between repo and local system...${fgcolor_reset}"

echo -e "\n${fgcolor_yellow_bold}=== [Neovim / LazyVim Configs] ===${fgcolor_reset}"
diff -ruN -x "lazy-lock.json" -x ".git" -x "example.lua" -x "README.md" -x "LICENSE" ./files/nvim "$HOME/.config/nvim" 2>/dev/null | $viewer || :

echo -e "\n${fgcolor_yellow_bold}=== [Ghostty Configs] ===${fgcolor_reset}"
diff -ruN ./files/ghostty "$HOME/.config/ghostty" 2>/dev/null | $viewer || :

echo -e "\n${fgcolor_yellow_bold}=== [Herdr Config] ===${fgcolor_reset}"
diff -u ./files/herdr/config.toml "$HOME/.config/herdr/config.toml" 2>/dev/null | $viewer || :

echo -e "\n${fgcolor_yellow_bold}=== [RTK Config] ===${fgcolor_reset}"
diff -u ./files/ai/rtk/config.toml "$HOME/.config/rtk/config.toml" 2>/dev/null | $viewer || :

echo -e "\n${fgcolor_yellow_bold}=== [Atuin Config] ===${fgcolor_reset}"
diff -u ./files/atuin/config.toml "$HOME/.config/atuin/config.toml" 2>/dev/null | $viewer || :

echo -e "\n${fgcolor_yellow_bold}=== [Starship Config] ===${fgcolor_reset}"
diff -u ./files/starship/starship.toml "$HOME/.config/starship.toml" 2>/dev/null | $viewer || :

echo -e "\n${fgcolor_yellow_bold}=== [Zsh Configs (.zshrc, .base.zsh, .tools.sh)] ===${fgcolor_reset}"
diff -u ./files/zsh/.zshrc "$HOME/.zshrc" 2>/dev/null | $viewer || :
diff -u ./files/zsh/.base.zsh "$HOME/.base.zsh" 2>/dev/null | $viewer || :
diff -u ./files/zsh/.tools.sh "$HOME/.tools.sh" 2>/dev/null | $viewer || :
diff -u ./files/zsh/.lscolors.sh "$HOME/.lscolors.sh" 2>/dev/null | $viewer || :

if [[ -d "$HOME/.config/Code/User" ]]; then
	echo -e "\n${fgcolor_yellow_bold}=== [VS Code Configs] ===${fgcolor_reset}"
	diff -u ./files/vscode/settings.json "$HOME/.config/Code/User/settings.json" 2>/dev/null | $viewer || :
	diff -u ./files/vscode/keybindings.json "$HOME/.config/Code/User/keybindings.json" 2>/dev/null | $viewer || :
fi

if [[ -d "$HOME/.config/Cursor/User" ]]; then
	echo -e "\n${fgcolor_yellow_bold}=== [Cursor IDE Configs] ===${fgcolor_reset}"
	diff -u ./files/cursor/settings.json "$HOME/.config/Cursor/User/settings.json" 2>/dev/null | $viewer || :
	diff -u ./files/cursor/keybindings.json "$HOME/.config/Cursor/User/keybindings.json" 2>/dev/null | $viewer || :
fi

echo -e "\n${fgcolor_green_bold}✔️ Diff inspection finished.${fgcolor_reset}"
