#!/bin/bash
# Run: ./utils/backup_config.sh
backup_storage_folder="$HOME/.arturonavax-env-backups"
backup_storage_max_size=$((50 * 1024 * 1024)) # 50MB
backup_storage_current_size=$(du -bs "$backup_storage_folder" | awk '{print $1}') || exit 1

backup_folder="$backup_storage_folder/$(date +'%Y-%m-%d_%H-%M')" || exit 1

mkdir -p "$backup_folder" || exit 1
mkdir -p "$backup_folder/snippets/" || exit 1
mkdir -p "$backup_folder/lsp-settings/" || exit 1
mkdir -p "$backup_folder/vscode/" || exit 1
mkdir -p "$backup_folder/lua/user" || exit 1

cp ~/.config/alacritty/alacritty.toml "$backup_folder/."

cp ~/.tmux.conf "$backup_folder/."

cp ~/.config/starship.toml "$backup_folder/."

cp ~/.lscolors.sh "$backup_folder/."
cp ~/.base.zsh "$backup_folder/."
cp ~/.tools.sh "$backup_folder/."

cp ~/.local/share/lunarvim/lvim/snapshots/default.json "$backup_folder/."
cp ~/.config/lvim/config.lua "$backup_folder/."
cp ~/.config/lvim/lua/user/* "$backup_folder/lua/user/."
cp ~/.config/lvim/snippets/* "$backup_folder/snippets/."
cp ~/.config/lvim/lsp-settings/* "$backup_folder/lsp-settings/."

if [[ "$(uname -s)" == "Linux" ]]; then
	cp ~/.config/Code/User/settings.json "$backup_folder/vscode/."
	cp ~/.config/Code/User/keybindings.json "$backup_folder/vscode/."

elif [[ "$(uname -s)" == "Darwin" ]]; then
	cp "$HOME/Library/Application Support/Code/User/settings.json" "$backup_folder/vscode/."
	cp "$HOME/Library/Application Support/Code/User/keybindings.json" "$backup_folder/vscode/."
fi

cp ~/.golangci.yml "$backup_folder/."
cp ~/.eslintrc.json "$backup_folder/."
cp ~/.prettierrc.json "$backup_folder/."
cp ~/.stylelintrc.json "$backup_folder/."
cp ~/.sql-formatter.json "$backup_folder/."

cp ~/.amethyst.yml "$backup_folder/."
cp ~/.config/karabiner/karabiner.json "$backup_folder/."

# Delete the oldest backup in case the container weight is more than 50MB.
while [ "$backup_storage_current_size" -gt "$backup_storage_max_size" ]; do
	oldest_backup=$(find "$backup_storage_folder" -mindepth 1 -maxdepth 1 -type d -printf '%T@ %p\n' |
		sort -n | head -n 1 | awk '{print $2}')

	rm -rf "$oldest_backup" && echo "The oldest backup ($oldest_backup) was deleted because the maximum backup size was exceeded."

	backup_storage_current_size=$(du -bs "$backup_storage_folder" | awk '{print $1}')
done
