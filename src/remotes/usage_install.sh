#!/bin/bash
# Check if the arguments are valid for the installation.
#
# Run: bash <(curl -fsSL "https://env.arturonavax.dev/usage_install.sh") "$@"
fcwb='\033[1;37m'
fcr='\033[0m'

editor="lvim"

function usage() {
	echo -e "$(
		cat <<EOF
[Installer]: List of installations:
  ${fcwb}requirements ${fcr}/ ${fcwb}r ${fcr}- Install the necessary tools, languages and dependencies.
  ${fcwb}fonts ${fcr}/ ${fcwb}f        ${fcr}- Install patched mono fonts.
  ${fcwb}terminal ${fcr}/ ${fcwb}t     ${fcr}- Install the terminal, shell, prompt, tmux and terminal tools.
  ${fcwb}editor ${fcr}/ ${fcwb}e       ${fcr}- Install the editor ($editor) and development tools.
  ${fcwb}osconfig ${fcr}/ ${fcwb}o     ${fcr}- Configure the operating system with personal preferences.
  ${fcwb}all ${fcr}/ ${fcwb}a          ${fcr}- Install and integrate all of the above.
  ${fcwb}help ${fcr}/ ${fcwb}h         ${fcr}- This helpful explanation.${fcr}
EOF
	)"
}

# check arguments
if [[ "$#" == 0 ]]; then
	usage

	exit 1
fi

for arg in "$@"; do
	case "$arg" in
	requirements | r) ;;
	fonts | f) ;;
	terminal | t) ;;
	editor | e) ;;
	osconfig | o) ;;
	all | a) ;;
	h | help)
		usage

		exit 1
		;;
	*)
		usage

		exit 1
		;;
	esac
done
