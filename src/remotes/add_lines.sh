#!/bin/bash
# Run: bash <(curl -fsSL "https://env.arturonavax.dev/add_lines.sh")
# get arguments
for arg in "$@"; do
	case "$arg" in
	base) add_base=1 ;;
	tools) add_tools=1 ;;
	basics) add_basics=1 ;;
	ngrok) add_ngrok=1 ;;
	esac
done

[[ "$#" == 0 ]] && add_basics=1

# ---

case "$SHELL" in
*zsh) shell_file=".zshrc" ;;
*tcsh) shell_file=".tcshrc" ;;
*ksh) shell_file=".kshrc" ;;
*csh) shell_file=".cshrc" ;;
*) shell_file=".bashrc" ;;
esac

touch ~/"$shell_file"

# add 'source ~/.base.zsh' command to the end of the file
if [[ "$add_base" == 1 ]]; then
	if [[ "$(command grep '^[^#]*\[\[ -f ~/\.base\.zsh \]\] && source ~/\.base\.zsh' ~/"$shell_file")" == "" ]]; then
		[[ "$(wc -l ~/"$shell_file" | awk '{print $1}')" != 0 ]] && echo >>~/"$shell_file"

		echo '# load base configuration' >>~/"$shell_file"
		echo '[[ -f ~/.base.zsh ]] && source ~/.base.zsh' >>~/"$shell_file"
	fi
fi

# add 'source ~/.tools.sh' command to the end of the file
if [[ "$add_tools" == 1 ]]; then
	if [[ "$(command grep '^[^#]*\[\[ -f ~/\.tools\.sh \]\] && source ~/\.tools\.sh' ~/"$shell_file")" == "" ]]; then
		[[ "$(wc -l ~/"$shell_file" | awk '{print $1}')" != 0 ]] && echo >>~/"$shell_file"

		echo '# load tools' >>~/"$shell_file"
		echo '[[ -f ~/.tools.sh ]] && source ~/.tools.sh' >>~/"$shell_file"
	fi
fi

# add the basic PATH only if they are not already there, and if the previous files have not been loaded. (~/.base.zsh and ~/.tools.sh)
if [[ "$add_basics" == 1 ]]; then
	# shellcheck disable=SC2016
	if [[ "$(command grep '^[^#]*\[\[ -f ~/\.tools\.sh \]\] && source ~/\.tools\.sh' ~/"$shell_file")" == "" &&
	"$(command grep '^[^#]*\[\[ -f ~/\.base\.zsh \]\] && source ~/\.base\.zsh' ~/"$shell_file")" == "" ]]; then
		if [[ "$(command grep '^[^#]*export PATH\=\"\$PATH:/usr/local/bin\"' ~/"$shell_file")" == "" &&
		"$(command grep '^[^#]*export PATH\=\"/usr/local/bin:\$PATH\"' ~/"$shell_file")" == "" ]]; then
			echo '[[ ":$PATH:" != *":/usr/local/bin:"* ]] && export PATH="$PATH:/usr/local/bin"' >>~/"$shell_file"
		fi

		if [[ "$(command grep '^[^#]*export PATH\=\"\$HOME/.local/bin:\$PATH\"' ~/"$shell_file")" == "" &&
		"$(command grep '^[^#]*export PATH\=\"\$PATH:\$HOME/.local/bin\"' ~/"$shell_file")" == "" ]]; then
			echo '[[ ":$PATH:" != *":$HOME/.local/bin:"* ]] && export PATH="$HOME/.local/bin:$PATH"' >>~/"$shell_file"
		fi
	fi
fi

# add 'console_ui_color: transparent' to the end of the file ngrok
if [[ "$add_ngrok" == 1 ]]; then
	if [[ "$(uname -s)" == "Linux" ]]; then
		ngrok_dirs=(
			"$HOME/.config/ngrok"
		)

		if [[ -d "$HOME/snap/ngrok/" ]]; then
			while read -r ngrok_dir; do
				ngrok_dirs+=("$ngrok_dir/.config/ngrok")
			done < <(command find ~/snap/ngrok -maxdepth 1 -type d -regex '.*/[0-9]+$')
		fi

		for ngrok_dir in "${ngrok_dirs[@]}"; do
			mkdir -p "$ngrok_dir"

			ngrok_file="$ngrok_dir/ngrok.yml"

			touch "$ngrok_file"

			if [[ "$(command grep '^[^#]*console_ui_color: transparent' "$ngrok_file")" == "" ]]; then
				echo 'console_ui_color: transparent' >>"$ngrok_file"
			fi
		done

	elif [[ "$(uname -s)" == "Darwin" ]]; then
		ngrok_dirs=(
			"$HOME/.config/ngrok"
			"$HOME/Library/Application Support/ngrok"
		)

		for ngrok_dir in "${ngrok_dirs[@]}"; do
			mkdir -p "$ngrok_dir"

			ngrok_file="$ngrok_dir/ngrok.yml"

			touch "$ngrok_file"

			if [[ "$(command grep '^[^#]*console_ui_color: transparent' "$ngrok_file")" == "" ]]; then
				echo 'console_ui_color: transparent' >>"$ngrok_file"
			fi
		done
	fi
fi
