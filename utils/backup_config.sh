#!/bin/bash
# Run: ./utils/backup_config.sh
backup_storage_folder="$HOME/.arturonavax-env-backups"
mkdir -p "$backup_storage_folder"
backup_storage_max_size=$((50 * 1024 * 1024)) # 50MB
backup_storage_current_size=$(du -bs "$backup_storage_folder" 2>/dev/null | awk '{print $1}')
backup_storage_current_size=${backup_storage_current_size:-0}

backup_folder="$backup_storage_folder/$(date +'%Y-%m-%d_%H-%M')"

mkdir -p "$backup_folder"
mkdir -p "$backup_folder/ghostty/"
mkdir -p "$backup_folder/nvim/"
mkdir -p "$backup_folder/vscode/"
mkdir -p "$backup_folder/cursor/"

if [[ -d ~/.config/ghostty ]]; then
	cp -r ~/.config/ghostty/* "$backup_folder/ghostty/." 2>/dev/null || :
fi

cp ~/.tmux.conf "$backup_folder/." 2>/dev/null || :

cp ~/.config/starship.toml "$backup_folder/." 2>/dev/null || :

cp ~/.lscolors.sh "$backup_folder/." 2>/dev/null || :
cp ~/.base.zsh "$backup_folder/." 2>/dev/null || :
cp ~/.tools.sh "$backup_folder/." 2>/dev/null || :

if [[ -d ~/.config/nvim ]]; then
	cp -r ~/.config/nvim/* "$backup_folder/nvim/." 2>/dev/null || :
fi

if [[ "$(uname -s)" == "Linux" ]]; then
	cp ~/.config/Code/User/settings.json "$backup_folder/vscode/." 2>/dev/null || :
	cp ~/.config/Code/User/keybindings.json "$backup_folder/vscode/." 2>/dev/null || :
	cp ~/.config/Cursor/User/settings.json "$backup_folder/cursor/." 2>/dev/null || :
	cp ~/.config/Cursor/User/keybindings.json "$backup_folder/cursor/." 2>/dev/null || :

elif [[ "$(uname -s)" == "Darwin" ]]; then
	cp "$HOME/Library/Application Support/Code/User/settings.json" "$backup_folder/vscode/." 2>/dev/null || :
	cp "$HOME/Library/Application Support/Code/User/keybindings.json" "$backup_folder/vscode/." 2>/dev/null || :
	cp "$HOME/Library/Application Support/Cursor/User/settings.json" "$backup_folder/cursor/." 2>/dev/null || :
	cp "$HOME/Library/Application Support/Cursor/User/keybindings.json" "$backup_folder/cursor/." 2>/dev/null || :
fi

cp ~/.golangci.yml "$backup_folder/." 2>/dev/null || :
cp ~/.eslintrc.json "$backup_folder/." 2>/dev/null || :
cp ~/.prettierrc.json "$backup_folder/." 2>/dev/null || :
cp ~/.stylelintrc.json "$backup_folder/." 2>/dev/null || :
cp ~/.sql-formatter.json "$backup_folder/." 2>/dev/null || :

cp ~/.amethyst.yml "$backup_folder/." 2>/dev/null || :
cp ~/.config/karabiner/karabiner.json "$backup_folder/." 2>/dev/null || :

# Delete the oldest backup in case the container weight is more than 50MB.
while [ "$backup_storage_current_size" -gt "$backup_storage_max_size" ]; do
	oldest_backup=$(find "$backup_storage_folder" -mindepth 1 -maxdepth 1 -type d -printf '%T@ %p\n' |
		sort -n | head -n 1 | awk '{print $2}')

	rm -rf "$oldest_backup" && echo "The oldest backup ($oldest_backup) was deleted because the maximum backup size was exceeded."

	backup_storage_current_size=$(du -bs "$backup_storage_folder" | awk '{print $1}')
done
