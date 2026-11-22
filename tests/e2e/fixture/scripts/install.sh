#!/bin/sh
set -eu

mkdir -p "$HOME/.local/bin"
cat >"$HOME/.local/bin/anx-e2e-tool" <<'SCRIPT'
#!/bin/sh
printf '%s\n' "environment smoke fixture"
SCRIPT
chmod +x "$HOME/.local/bin/anx-e2e-tool"
printf '%s\n' "installed" >>"$HOME/.anx-e2e-installs"
