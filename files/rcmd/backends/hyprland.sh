#!/usr/bin/env bash
# ==============================================================================
# rcmd backend: Hyprland (Wayland)
# Uses hyprctl to query and focus windows with MRU ordering, cycling, and title match.
# Cleans desktop prefixes (gnome-, xfce4-, kde-) in dynamic mode.
# ==============================================================================

rcmd_backend_hyprland() {
    local key="${1,,}"
    local cmd="$2"
    local pattern="$3"
    local match_mode="${4:-class}"

    if ! command -v hyprctl >/dev/null 2>&1; then
        [ -n "$cmd" ] && nohup bash -c "$cmd" >/dev/null 2>&1 &
        return 0
    fi

    python3 -c "
import json, subprocess, sys, re

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

matched = []
if pattern:
    if match_mode == 'title':
        matched = [c for c in clients if pattern in c.get('title', '').lower()]
    else:
        matched = [c for c in clients if pattern in c.get('class', '').lower() or pattern in c.get('title', '').lower()]
elif key:
    # Dynamic fallback: ONLY currently open apps starting with that letter (lowercase)
    sorted_clients = sorted(clients, key=lambda c: c.get('focusHistoryID', 999))
    matched_class = ''
    for c in sorted_clients:
        raw_cls = c.get('class', '').lower()
        title = c.get('title', '').lower()
        clean_cls = re.sub(r'^(gnome-|xfce4-|xfce-|kde-|org\.[^.]+\.|com\.[^.]+\.)', '', raw_cls)
        candidates = [raw_cls, clean_cls]
        if ' - ' in title or ' — ' in title:
            parts = re.split(r' [—\-] ', title)
            candidates.append(parts[-1].strip())
        else:
            clean_title = re.sub(r'^\([0-9]+\+?\)[[:space:]]*', '', title)
            candidates.extend(clean_title.split())

        if any(cand.startswith(key) for cand in candidates if cand):
            matched_class = c.get('class')
            break
    if matched_class:
        matched = [c for c in clients if c.get('class') == matched_class]
    else:
        sys.exit(0)

if not matched:
    if cmd:
        subprocess.Popen(cmd, shell=True)
    sys.exit(0)

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
