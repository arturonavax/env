#!/bin/bash
# This script copies all the configuration files to this repository.
# This script is made to run from the repository root.
#
# Run: bash ./utils/copy_config.sh
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

# Copy Terminal files
mkdir -p ./files/ghostty/auto/
[[ -f ~/.config/ghostty/config ]] && cp ~/.config/ghostty/config ./files/ghostty/.
[[ -f ~/.config/ghostty/auto/theme.ghostty ]] && cp ~/.config/ghostty/auto/theme.ghostty ./files/ghostty/auto/.

cp ~/.tmux.conf ./files/tmux/.

cp ~/.config/starship.toml ./files/starship/.

cp ~/.lscolors.sh ./files/zsh/.
cp ~/.base.zsh ./files/zsh/.
cp ~/.tools.sh ./files/zsh/.

# Copy Editor (LazyVim) files - ONLY customizable and transferable configs
mkdir -p ./files/nvim/lua/config/ ./files/nvim/lua/plugins/

[[ -f ~/.config/nvim/init.lua ]] && cp ~/.config/nvim/init.lua ./files/nvim/.
[[ -f ~/.config/nvim/lazyvim.json ]] && cp ~/.config/nvim/lazyvim.json ./files/nvim/.
[[ -f ~/.config/nvim/stylua.toml ]] && cp ~/.config/nvim/stylua.toml ./files/nvim/.
[[ -f ~/.config/nvim/.gitignore ]] && cp ~/.config/nvim/.gitignore ./files/nvim/.
[[ -f ~/.config/nvim/.neoconf.json ]] && cp ~/.config/nvim/.neoconf.json ./files/nvim/.
[[ -f ~/.config/nvim/lazy-lock.json ]] && cp ~/.config/nvim/lazy-lock.json ./files/nvim/.

# Core Lua configs
if [[ -d ~/.config/nvim/lua/config ]]; then
	cp ~/.config/nvim/lua/config/*.lua ./files/nvim/lua/config/. 2>/dev/null || :
fi

# User plugin specs (excluding example.lua template)
if [[ -d ~/.config/nvim/lua/plugins ]]; then
	for plugin_file in ~/.config/nvim/lua/plugins/*.lua; do
		if [[ -f "$plugin_file" && "$(basename "$plugin_file")" != "example.lua" ]]; then
			cp "$plugin_file" ./files/nvim/lua/plugins/.
		fi
	done
fi

# Copy VSCode files
mkdir -p ./files/vscode/
if [[ "$(uname -s)" == "Linux" ]]; then
	[[ -f ~/.config/Code/User/settings.json ]] && cp ~/.config/Code/User/settings.json ./files/vscode/.
	[[ -f ~/.config/Code/User/keybindings.json ]] && cp ~/.config/Code/User/keybindings.json ./files/vscode/.
elif [[ "$(uname -s)" == "Darwin" ]]; then
	[[ -f "$HOME/Library/Application Support/Code/User/settings.json" ]] && cp "$HOME/Library/Application Support/Code/User/settings.json" ./files/vscode/.
	[[ -f "$HOME/Library/Application Support/Code/User/keybindings.json" ]] && cp "$HOME/Library/Application Support/Code/User/keybindings.json" ./files/vscode/.
fi

# Copy Cursor files
mkdir -p ./files/cursor/
if [[ "$(uname -s)" == "Linux" ]]; then
	[[ -f ~/.config/Cursor/User/settings.json ]] && cp ~/.config/Cursor/User/settings.json ./files/cursor/.
	[[ -f ~/.config/Cursor/User/keybindings.json ]] && cp ~/.config/Cursor/User/keybindings.json ./files/cursor/.
elif [[ "$(uname -s)" == "Darwin" ]]; then
	[[ -f "$HOME/Library/Application Support/Cursor/User/settings.json" ]] && cp "$HOME/Library/Application Support/Cursor/User/settings.json" ./files/cursor/.
	[[ -f "$HOME/Library/Application Support/Cursor/User/keybindings.json" ]] && cp "$HOME/Library/Application Support/Cursor/User/keybindings.json" ./files/cursor/.
fi

cp ~/.golangci.yml ./files/golangci-lint/.
cp ~/.eslintrc.json ./files/eslint/.
cp ~/.prettierrc.json ./files/prettier/.
cp ~/.stylelintrc.json ./files/stylelint/.
cp ~/.sql-formatter.json ./files/sql-formatter/.

[[ -f ~/.amethyst.yml ]] && cp ~/.amethyst.yml ./files/amethyst/. || :
[[ -f ~/.config/karabiner/karabiner.json ]] && cp ~/.config/karabiner/karabiner.json ./files/karabiner/. || :

echo "Configuration files successfully copied to repository!"
