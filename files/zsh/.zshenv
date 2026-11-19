if [ -n "${XDG_RUNTIME_DIR:-}" ] && [ "$(uname -s 2>/dev/null)" = "Linux" ]; then
    export SSH_AUTH_SOCK="${XDG_RUNTIME_DIR}/ssh-agent.socket"
fi
