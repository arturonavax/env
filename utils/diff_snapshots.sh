#!/bin/bash
# This script compares the snapshot commits file of this repo with the official LunarVim one.
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

my_snapshots_file="./files/lvim/default.json"
official_snapshots_file="$HOME/.local/share/lunarvim/lvim/snapshots/default.json"
official_snapshots_site="https://raw.githubusercontent.com/LunarVim/LunarVim/master/snapshots/default.json"

# set viewer
if [[ "$(command -v batcat)" != "" ]]; then
	viewer="batcat"

elif [[ "$(command -v bat)" != "" ]]; then
	viewer="bat"

elif [[ "$(command -v less)" != "" ]]; then
	viewer="less"

elif [[ "$(command -v more)" != "" ]]; then
	viewer="more"

elif [[ "$(command -v cat)" != "" ]]; then
	viewer="cat"
fi

temp_file1="$(mktemp)"
temp_file2="$(mktemp)"

curl -fsSL "$official_snapshots_site" | jq --tab >"$temp_file1"

jq --tab <"$my_snapshots_file" >"$temp_file2"

if [[ "$(command -v wdiff)" != "" ]]; then
	differentiator="wdiff"

elif [[ "$(command -v lvim)" != "" ]]; then
	differentiator="lvim -d"

elif [[ "$(command -v nvim)" != "" ]]; then
	differentiator="nvim -d"

elif [[ "$(command -v vim)" != "" ]]; then
	differentiator="vim -d"

elif [[ "$(command -v diff)" != "" ]]; then
	differentiator="diff"
fi

eval "$differentiator $temp_file1 $temp_file2 | $viewer"

if ! diff -q "$temp_file1" "$temp_file2" >/dev/null; then
	echo -n "Apply official changes? (y/n): "

	read -r reply

	reply=$(echo "$reply" | tr '[:upper:]' '[:lower:]')

	if [[ "$reply" == "s" || "$reply" == "y" || "$reply" == "si" || "$reply" == "yes" ]]; then
		curl -fsSL 'https://raw.githubusercontent.com/LunarVim/LunarVim/master/snapshots/default.json' |
			jq --tab |
			tee "$my_snapshots_file" >"$official_snapshots_file"
	fi

fi
