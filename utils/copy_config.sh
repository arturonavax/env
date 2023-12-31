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

# Copy files
cp ~/.config/alacritty/alacritty.toml ./files/alacritty/.
cp ~/.config/alacritty/dark_theme.toml ./files/alacritty/.
cp ~/.config/alacritty/light_theme.toml ./files/alacritty/.

cp ~/.tmux.conf ./files/tmux/.

cp ~/.config/starship.toml ./files/starship/.

cp ~/.lscolors.sh ./files/zsh/.
cp ~/.base.zsh ./files/zsh/.
cp ~/.tools.sh ./files/zsh/.

[[ "$(command -v jq)" != "" ]] && jq --tab <"$HOME/.local/share/lunarvim/lvim/snapshots/default.json" >./files/lvim/default.json
cp ~/.config/lvim/config.lua ./files/lvim/.
cp ~/.config/lvim/lua/user/* ./files/lvim/lua/user/.
cp ~/.config/lvim/snippets/* ./files/snippets/.
cp ~/.config/lvim/lsp-settings/* ./files/lsp-settings/.

if [[ "$(uname -s)" == "Linux" ]]; then
	cp ~/.config/Code/User/settings.json ./files/vscode/.
	cp ~/.config/Code/User/keybindings.json ./files/vscode/.

elif [[ "$(uname -s)" == "Darwin" ]]; then
	cp "$HOME/Library/Application Support/Code/User/settings.json" ./files/vscode/.
	cp "$HOME/Library/Application Support/Code/User/keybindings.json" ./files/vscode/.
fi

cp ~/.golangci.yml ./files/golangci-lint/.
cp ~/.eslintrc.json ./files/eslint/.
cp ~/.prettierrc.json ./files/prettier/.
cp ~/.stylelintrc.json ./files/stylelint/.
cp ~/.sql-formatter.json ./files/sql-formatter/.

cp ~/.amethyst.yml ./files/amethyst/.
