#!/usr/bin/env bash
set -euo pipefail

bin_dir=${1:?Usage: install_hledger.sh /absolute/path/to/bin}
tmp_dir=$(mktemp -d)
trap 'rm -rf -- "$tmp_dir"' EXIT

# Ignore user Eget settings so ANX always extracts the stable release into staging.
touch "$tmp_dir/eget.toml"
# Eget can interpret a repository shorthand as a path relative to its working directory.
(
	cd "$tmp_dir"
	EGET_CONFIG="$tmp_dir/eget.toml" eget hledgerorg/hledger --all --to "$tmp_dir" </dev/null
)

# Publish the catalog's primary binary last, after validating all three files.
binaries=(hledger-ui hledger-web hledger)
for binary in "${binaries[@]}"; do
	if [[ ! -f $tmp_dir/$binary || ! -s $tmp_dir/$binary ]]; then
		echo "hledger release did not contain $binary" >&2
		exit 1
	fi
done

mkdir -p "$bin_dir"
for binary in "${binaries[@]}"; do
	install -m 0755 "$tmp_dir/$binary" "$bin_dir/$binary"
done
