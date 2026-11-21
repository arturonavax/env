_anx_path_prepend() {
    [ -n "$1" ] && [ -d "$1" ] || return 0
    case ":${PATH}:" in
        *":$1:"*) ;;
        *) PATH="$1:${PATH}" ;;
    esac
}

_anx_path_prepend "/usr/local/go/bin"
_anx_path_prepend "/opt/homebrew/sbin"
_anx_path_prepend "/opt/homebrew/bin"
if [ -n "${HOME:-}" ]; then
    _anx_path_prepend "${HOME}/.pyenv/shims"
    _anx_path_prepend "${HOME}/go/bin"
    _anx_path_prepend "${HOME}/.cargo/bin"
    _anx_path_prepend "${HOME}/.local/bin"
fi
export PATH
unset -f _anx_path_prepend 2>/dev/null || true

if [ -n "${XDG_RUNTIME_DIR:-}" ] && [ "$(uname -s 2>/dev/null)" = "Linux" ]; then
    export SSH_AUTH_SOCK="${XDG_RUNTIME_DIR}/ssh-agent.socket"
fi
