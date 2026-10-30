#!/bin/bash
#   This file echoes a bunch of 24-bit color codes
#   to the terminal to demonstrate its functionality.
#   The foreground escape sequence is ^[38;2;<r>;<g>;<b>m
#   The background escape sequence is ^[48;2;<r>;<g>;<b>m
#   <r> <g> <b> range from 0 to 255 inclusive.
#   The escape sequence ^[0m returns output to default
#
# Run: curl -fsSL "https://env.arturonavax.dev/24-bit-colors.sh" | bash
setBackgroundColor() {
	echo -en "\x1b[48;2;$1;$2;$3""m"
}

export -f setBackgroundColor

resetOutput() {
	echo -en "\x1b[0m\n"
}

# Gives a color $1/255 % along HSV
# Who knows what happens when $1 is outside 0-255
# Echoes "$red $green $blue" where
# $red $green and $blue are integers
# ranging between 0 and 255 inclusive
rainbowColor() {
	h=$(("$1" / 43))
	f=$(("$1" - 43 * "$h"))
	t=$(("$f" * 255 / 43))
	q=$((255 - t))

	case "$h" in
	0) echo "255 $t 0" ;;

	1) echo "$q 255 0" ;;

	2) echo "0 255 $t" ;;

	3) echo "0 $q 255" ;;

	4) echo "$t 0 255" ;;

	5) echo "255 0 $q" ;;

	*) echo "0 0 0" ;;
	esac
}

seq 0 127 | while read -r line; do
	setBackgroundColor "$line" 0 0

	echo -en " "
done

resetOutput

seq 255 128 | while read -r line; do
	setBackgroundColor "$line" 0 0

	echo -en " "
done

resetOutput

seq 0 127 | while read -r line; do
	setBackgroundColor 0 "$line" 0

	echo -n " "
done

resetOutput

seq 255 128 | while read -r line; do
	setBackgroundColor 0 "$line" 0

	echo -n " "
done

resetOutput

seq 0 127 | while read -r line; do
	setBackgroundColor 0 0 "$line"

	echo -n " "
done

resetOutput

seq 255 128 | while read -r line; do
	setBackgroundColor 0 0 "$line"

	echo -n " "
done

resetOutput

seq 0 127 | while read -r line; do
	rainbowColor "$line" | xargs -I {} bash -c "setBackgroundColor {}"

	echo -n " "
done

resetOutput

seq 255 128 | while read -r line; do
	rainbowColor "$line" | xargs -I {} bash -c "setBackgroundColor {}"

	echo -n " "
done

resetOutput
