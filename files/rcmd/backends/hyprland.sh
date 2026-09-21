#!/usr/bin/env bash
# ==============================================================================
# rcmd backend: Hyprland (Wayland)
# Uses hyprctl to query and focus windows with MRU ordering, cycling, and title match.
# ==============================================================================

rcmd_backend_hyprland() {
    local key="$1"
    local cmd="$2"
    local pattern="$3"
    local match_mode="${4:-class}"

    if ! command -v hyprctl >/dev/null 2>&1; then
        [ -n "$cmd" ] && nohup bash -c "$cmd" >/dev/null 2>&1 &
        return 0
    fi

    python3 -c "
import json, subprocess, sys

key = sys.argv[1].lower()
cmd = sys.argv[2]
pattern = sys.argv[3].lower()
match_mode = sys.argv[4].lower()

try:
    clients_raw = subprocess.check_output(['hyprctl', 'clients', '-j']).decode('utf-8')
    clients = json.loads(clients_raw)
except Exception:
    clients = []

try:
    active_raw = subprocess.check_output(['hyprctl', 'activewindow', '-j']).decode('utf-8')
    active = json.loads(active_raw)
    active_addr = active.get('address', '')
except Exception:
    active_addr = ''

# Filter matching windows
matched = []
if pattern:
    if match_mode == 'title':
        matched = [c for c in clients if pattern in c.get('title', '').lower()]
    else:
        matched = [c for c in clients if pattern in c.get('class', '').lower() or pattern in c.get('title', '').lower()]
elif key:
    # Dynamic fallback: find first client starting with key
    sorted_clients = sorted(clients, key=lambda c: c.get('focusHistoryID', 999))
    matched_class = ''
    for c in sorted_clients:
        cls = c.get('class', '').lower()
        title = c.get('title', '').lower()
        if cls.startswith(key) or title.startswith(key):
            matched_class = cls
            break
    if matched_class:
        matched = [c for c in clients if c.get('class', '').lower() == matched_class]

if not matched:
    if cmd:
        subprocess.Popen(cmd, shell=True)
    sys.exit(0)

# Check if active window is in matched
addrs = [c.get('address') for c in matched]
if active_addr in addrs:
    curr_idx = addrs.index(active_addr)
    next_idx = (curr_idx + 1) % len(addrs)
    target_addr = addrs[next_idx]
else:
    matched.sort(key=lambda c: c.get('focusHistoryID', 999))
    target_addr = matched[0].get('address')

if target_addr:
    subprocess.run(['hyprctl', 'dispatch', 'focuswindow', f'address:{target_addr}'], check=False)
" "$key" "$cmd" "$pattern" "$match_mode"
}
