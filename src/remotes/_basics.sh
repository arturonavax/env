#!/bin/bash
# This script must be executed with the "source" command.
#
# Run: source <(curl -fsSL "https://env.arturonavax.dev/_basics.sh" | cat)
mkdir -p "$HOME/.local/bin/"
sudo mkdir -m 755 -p /usr/local/bin # there are MacOS systems that do not have this folder

# Basics
# shellcheck disable=SC2015
[[ ":$PATH:" != *":/usr/local/bin:"* ]] && export PATH="$PATH:/usr/local/bin" || :
# shellcheck disable=SC2015
[[ ":$PATH:" != *":$HOME/.local/bin:"* ]] && export PATH="$HOME/.local/bin:$PATH" || :
