#!/bin/bash
# Run: ./utils/backup_config.sh
mkdir -p ~/.arturonavax-env-backups/snippets/
mkdir -p ~/.arturonavax-env-backups/lsp-settings/
mkdir -p ~/.arturonavax-env-backups/vscode/
mkdir -p ~/.arturonavax-env-backups/lua/user
mkdir -p ~/.arturonavax-env-backups/

cp ~/.config/alacritty/alacritty.yml ~/.arturonavax-env-backups/.

cp ~/.tmux.conf ~/.arturonavax-env-backups/.

cp ~/.config/starship.toml ~/.arturonavax-env-backups/.

cp ~/.lscolors.sh ~/.arturonavax-env-backups/.
cp ~/.base.zsh ~/.arturonavax-env-backups/.
cp ~/.tools.sh ~/.arturonavax-env-backups/.

cp ~/.local/share/lunarvim/lvim/snapshots/default.json ~/.arturonavax-env-backups/.
cp ~/.config/lvim/config.lua ~/.arturonavax-env-backups/.
cp ~/.config/lvim/lua/user/* ~/.arturonavax-env-backups/lua/user/.
cp ~/.config/lvim/snippets/* ~/.arturonavax-env-backups/snippets/.
cp ~/.config/lvim/lsp-settings/* ~/.arturonavax-env-backups/lsp-settings/.

if [[ "$(uname -s)" == "Linux" ]]; then
	cp ~/.config/Code/User/settings.json ~/.arturonavax-env-backups/vscode/.
	cp ~/.config/Code/User/keybindings.json ~/.arturonavax-env-backups/vscode/.

elif [[ "$(uname -s)" == "Darwin" ]]; then
	cp "$HOME/Library/Application Support/Code/User/settings.json" ~/.arturonavax-env-backups/vscode/.
	cp "$HOME/Library/Application Support/Code/User/keybindings.json" ~/.arturonavax-env-backups/vscode/.
fi

cp ~/.golangci.yml ~/.arturonavax-env-backups/.
cp ~/.eslintrc.json ~/.arturonavax-env-backups/.
cp ~/.prettierrc.json ~/.arturonavax-env-backups/.
cp ~/.stylelintrc.json ~/.arturonavax-env-backups/.
cp ~/.sql-formatter.json ~/.arturonavax-env-backups/.
