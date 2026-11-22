#!/usr/bin/env bash
set -euo pipefail

repo=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf -- "$tmp"' EXIT
export HOME="$tmp/home" TMPDIR="$tmp"
mkdir -p "$HOME" "$tmp/stubs"
export PATH="$tmp/stubs:$PATH"

cat >"$tmp/stubs/curl" <<'EOF'
#!/usr/bin/env bash
while [[ $1 != -o ]]; do shift; done
[[ $3 == https://zyedidia.github.io/eget.sh ]] || exit 1
cat >"$2" <<'SCRIPT'
printf '#!/usr/bin/env bash\n' >eget
touch "$HOME/bootstrap-ran"
SCRIPT
EOF
cat >"$tmp/stubs/sha256sum" <<'EOF'
#!/usr/bin/env bash
read -r digest file
[[ $digest == 0e64b8a3c13f531da005096cc364ac77835bda54276fedef6c62f3dbdc1ee919 &&
	$file == eget.sh && -s $file && ${BAD_HASH:-0} == 0 ]]
EOF
chmod +x "$tmp/stubs/"*

dest="$HOME/bin with spaces"
if BAD_HASH=1 bash "$repo/scripts/install_eget.sh" "$dest"; then
	echo 'Expected the bootstrap to reject a mismatched checksum' >&2
	exit 1
fi
[[ ! -e $HOME/bootstrap-ran && ! -e $dest/eget ]]
bash "$repo/scripts/install_eget.sh" "$dest"
[[ -x $dest/eget && -f $HOME/bootstrap-ran ]]

cat >"$tmp/stubs/eget" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[[ $# == 4 && $1 == hledgerorg/hledger && $2 == --all && $3 == --to ]]
[[ $PWD == "$4" ]]
[[ -f $EGET_CONFIG && ! -s $EGET_CONFIG ]]
if [[ ${FAIL_DOWNLOAD:-0} == 1 ]]; then
	exit 23
fi
for binary in hledger hledger-ui hledger-web; do
	if [[ ${MISSING_BINARY:-} != "$binary" ]]; then
		printf '#!/usr/bin/env bash\n' >"$4/$binary"
	fi
done
EOF
chmod +x "$tmp/stubs/eget"

if MISSING_BINARY=hledger-web bash "$repo/scripts/install_hledger.sh" "$dest"; then
	echo 'Expected an incomplete release to fail' >&2
	exit 1
fi
[[ ! -e $dest/hledger && ! -e $dest/hledger-ui && ! -e $dest/hledger-web ]]
bash "$repo/scripts/install_hledger.sh" "$dest"
for binary in hledger hledger-ui hledger-web; do
	[[ -x $dest/$binary ]]
	printf 'existing %s\n' "$binary" >"$dest/$binary"
done

status=0
FAIL_DOWNLOAD=1 bash "$repo/scripts/install_hledger.sh" "$dest" || status=$?
[[ $status == 23 ]]
for binary in hledger hledger-ui hledger-web; do
	[[ -x $dest/$binary && $(<"$dest/$binary") == "existing $binary" ]]
done
echo 'hledger installer checks passed'
