#!/bin/bash
# This script fixes current problems in the
#
# Run: curl -fsSL "https://env.arturonavax.dev/fixer.sh" | bash
# function fixer() {
# 	# LunarVim - NvimTree.lua: Fix file system upgrade problems.
# 	nvimtree_folder="$HOME/.local/share/lunarvim/site/pack/lazy/opt/nvim-tree.lua/"
# 	nvimtree_commit_fixer="270c95556cad96d18ca547d86ae65927334b108b"

# 	[[ -d "$nvimtree_folder" ]] &&
# 		cd "$nvimtree_folder" &&
# 		git checkout "$nvimtree_commit_fixer"
# }

lines="$(declare -f fixer)"

count_lines="$(echo "$lines" | grep -c -v '^[[:space:]]*$')"

if (("$count_lines" >= 3)); then
	fixer

	echo "!!! The fixer.sh script is doing things, make sure that it is still needed."
fi
