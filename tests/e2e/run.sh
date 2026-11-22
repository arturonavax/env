#!/bin/sh
set -eu

if [ "$#" -ne 2 ]; then
    printf 'Usage: %s /path/to/anx-linux <local-docker-image>\n' "$0" >&2
    exit 2
fi

binary_dir=$(CDPATH= cd -- "$(dirname -- "$1")" && pwd -P)
binary="$binary_dir/$(basename -- "$1")"
repo=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd -P)
image=$2
socket=${ANX_E2E_DOCKER_SOCKET:-/var/run/docker.sock}
case "$image" in -*) printf 'Invalid image name\n' >&2; exit 2 ;; esac
case "$socket" in /*) ;; *) printf 'Docker socket must be an absolute local path\n' >&2; exit 2 ;; esac

if [ ! -f "$binary" ] || [ ! -x "$binary" ]; then
    printf 'Expected an executable Linux ANX binary at %s\n' "$binary" >&2
    exit 2
fi
if ! docker --host "unix://$socket" image inspect "$image" >/dev/null; then
    printf 'Use a Docker image that is already available locally: %s\n' "$image" >&2
    exit 2
fi

docker --host "unix://$socket" run --rm --pull=never --network=none --read-only \
    --cap-drop=ALL --security-opt=no-new-privileges \
    --user "$(id -u):$(id -g)" --tmpfs /tmp:rw,exec,nosuid,nodev,mode=1777 \
    --volume "$repo:/env:ro" --volume "$binary:/opt/anx:ro" \
    --env HOME=/tmp/anx-home --env XDG_CONFIG_HOME=/tmp/anx-home/config \
    --env XDG_STATE_HOME=/tmp/anx-home/state --env XDG_CACHE_HOME=/tmp/anx-home/cache \
    --env NO_COLOR=1 --env "ANX_E2E_PROFILE=${ANX_E2E_PROFILE:-}" \
    --workdir /tmp --entrypoint /bin/sh "$image" -eu -c '
        mkdir -p "$HOME"
        export PATH="$HOME/.local/bin:$PATH"

        export ANX_CONFIG_REPO=/env
        /opt/anx repo verify --offline
        if [ -n "$ANX_E2E_PROFILE" ]; then
            /opt/anx plan --offline --profile "$ANX_E2E_PROFILE"
        fi

        export ANX_CONFIG_REPO=/env/tests/e2e/fixture
        /opt/anx repo verify --offline
        /opt/anx apply --yes --offline --profile container-smoke
        test "$(anx-e2e-tool)" = "environment smoke fixture"
        grep -Fx "environment fixture config" "$HOME/.config/anx-e2e.conf"
        distro=$(sed -n "s/^ID=//p" /etc/os-release | tr -d "\042")
        if [ "$distro" = "alpine" ]; then
            grep -Fx "environment fixture config" "$HOME/.config/anx-e2e-alpine.conf"
        else
            test ! -e "$HOME/.config/anx-e2e-alpine.conf"
        fi

        /opt/anx apply --yes --offline --profile container-smoke
        test "$(wc -l < "$HOME/.anx-e2e-installs")" -eq 1
        printf "%s\n" "Environment smoke test passed, including idempotent package installation."
    '
