#!/bin/bash
# This script inspects differences between the repo configuration files and the local system.
# Run: ./utils/diff_snapshots.sh
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

# set viewer
if [[ "$(command -v batcat)" != "" ]]; then
	viewer="batcat"

elif [[ "$(command -v bat)" != "" ]]; then
	viewer="bat"

elif [[ "$(command -v less)" != "" ]]; then
	viewer="less"

else
	viewer="cat"
fi

if [[ "$(command -v wdiff)" != "" ]]; then
	differentiator="wdiff"

elif [[ "$(command -v diff)" != "" ]]; then
	differentiator="diff -u"
fi

echo -e "\033[1;37m[Diff Config]: Checking differences between repo and local system...\033[0m"

echo -e "\n\033[1;33m=== [Neovim / LazyVim Configs] ===\033[0m"
diff -ruN -x "lazy-lock.json" -x ".git" -x "example.lua" -x "README.md" -x "LICENSE" ./files/nvim "$HOME/.config/nvim" 2>/dev/null | $viewer || :

echo -e "\n\033[1;33m=== [Ghostty Configs] ===\033[0m"
diff -ruN ./files/ghostty "$HOME/.config/ghostty" 2>/dev/null | $viewer || :

echo -e "\n\033[1;33m=== [Tmux Config] ===\033[0m"
diff -u ./files/tmux/.tmux.conf "$HOME/.tmux.conf" 2>/dev/null | $viewer || :

echo -e "\n\033[1;33m=== [Starship Config] ===\033[0m"
diff -u ./files/starship/starship.toml "$HOME/.config/starship.toml" 2>/dev/null | $viewer || :

echo -e "\n\033[1;33m=== [Zsh Base Config] ===\033[0m"
diff -u ./files/zsh/.base.zsh "$HOME/.base.zsh" 2>/dev/null | $viewer || :

if [[ -d "$HOME/.config/Code/User" ]]; then
	echo -e "\n\033[1;33m=== [VS Code Configs] ===\033[0m"
	diff -u ./files/vscode/settings.json "$HOME/.config/Code/User/settings.json" 2>/dev/null | $viewer || :
	diff -u ./files/vscode/keybindings.json "$HOME/.config/Code/User/keybindings.json" 2>/dev/null | $viewer || :
fi

if [[ -d "$HOME/.config/Cursor/User" ]]; then
	echo -e "\n\033[1;33m=== [Cursor IDE Configs] ===\033[0m"
	diff -u ./files/cursor/settings.json "$HOME/.config/Cursor/User/settings.json" 2>/dev/null | $viewer || :
	diff -u ./files/cursor/keybindings.json "$HOME/.config/Cursor/User/keybindings.json" 2>/dev/null | $viewer || :
fi

echo -e "\n\033[1;32m✔️ Diff inspection finished.\033[0m"
