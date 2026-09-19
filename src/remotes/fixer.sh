#!/bin/bash
# This script fixes current problems in the
#
# Run: curl -fsSL "https://env.arturonavax.dev/fixer.sh" | bash


lines="$(declare -f fixer)"

count_lines="$(echo "$lines" | grep -c -v '^[[:space:]]*$')"

if (("$count_lines" >= 3)); then
	fixer

	echo "!!! The fixer.sh script is doing things, make sure that it is still needed."
fi
