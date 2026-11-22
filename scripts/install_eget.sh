#!/usr/bin/env bash
set -euo pipefail

bin_dir=${1:?Usage: install_eget.sh /absolute/path/to/bin}
tmp_dir=$(mktemp -d)
trap 'rm -rf -- "$tmp_dir"' EXIT

# Published by upstream: https://github.com/zyedidia/eget#quick-install-script
sha256=0e64b8a3c13f531da005096cc364ac77835bda54276fedef6c62f3dbdc1ee919
(
	cd "$tmp_dir"
	curl -fsSL --retry 3 --connect-timeout 15 -o eget.sh https://zyedidia.github.io/eget.sh
	printf '%s  eget.sh\n' "$sha256" | sha256sum --check -
	bash ./eget.sh
	mkdir -p "$bin_dir"
	install -m 0755 eget "$bin_dir/eget"
)
