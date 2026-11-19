#!/usr/bin/env bash
# ==============================================================================
# rcmd backend: Sway / i3 (wlroots / Wayland)
# Uses swaymsg to query and focus containers with dynamic matching and title support.
# Cleans desktop prefixes (gnome-, xfce4-, kde-) in dynamic mode.
# ==============================================================================

rcmd_backend_sway() {
    local key="${1,,}"
    local cmd="$2"
    local pattern="$3"
    local match_mode="${4:-class}"

    local msg_cmd="swaymsg"
    command -v swaymsg >/dev/null 2>&1 || msg_cmd="i3-msg"

    if ! command -v "$msg_cmd" >/dev/null 2>&1; then
        [ -n "$cmd" ] && nohup bash -c "$cmd" >/dev/null 2>&1 &
        return 0
    fi

    python3 -c "
import json, subprocess, sys, re

key = sys.argv[1].lower()
cmd = sys.argv[2]
pattern = sys.argv[3].lower()
match_mode = sys.argv[4].lower()
msg_cmd = sys.argv[5]

try:
    tree = json.loads(subprocess.check_output([msg_cmd, '-t', 'get_tree']).decode('utf-8'))
except Exception:
    tree = {}

def get_nodes(node):
    res = []
    if node.get('nodes'):
        for n in node['nodes']:
            res.extend(get_nodes(n))
    if node.get('floating_nodes'):
        for n in node['floating_nodes']:
            res.extend(get_nodes(n))
    app_id = (node.get('app_id') or '').lower()
    wp = node.get('window_properties') or {}
    cls = (wp.get('class') or '').lower()
    name = (node.get('name') or '').lower()
    if app_id or cls or name:
        res.append({
            'id': node.get('id'),
            'app_id': app_id,
            'class': cls,
            'name': name,
            'focused': node.get('focused', False)
        })
    return res

nodes = get_nodes(tree)
matched = []
if pattern:
    if match_mode == 'title':
        matched = [n for n in nodes if pattern in n['name']]
    else:
        matched = [n for n in nodes if pattern in n['app_id'] or pattern in n['class']]
elif key:
    # Dynamic fallback: ONLY currently open apps starting with that letter (lowercase)
    target_cls = ''
    for n in nodes:
        raw_app = n['app_id'] or n['class']
        clean_app = re.sub(r'^(gnome-|xfce4-|xfce-|kde-|org\.[^.]+\.|com\.[^.]+\.)', '', raw_app)
        candidates = [raw_app, clean_app]
        title = n['name']
        if ' - ' in title or ' — ' in title:
            parts = re.split(r' [—\-] ', title)
            candidates.append(parts[-1].strip())
        else:
            clean_title = re.sub(r'^\([0-9]+\+?\)[[:space:]]*', '', title)
            candidates.extend(clean_title.split())

        if any(cand.startswith(key) for cand in candidates if cand):
            target_cls = n['app_id'] or n['class']
            break
    if target_cls:
        matched = [n for n in nodes if n['app_id'].lower() == target_cls or n['class'].lower() == target_cls]
    else:
        sys.exit(0)

if not matched:
    if cmd:
        subprocess.Popen(f'{msg_cmd} exec \"{cmd}\"', shell=True)
    sys.exit(0)

focused_in_matched = [n for n in matched if n['focused']]
if focused_in_matched:
    curr_idx = matched.index(focused_in_matched[0])
    next_node = matched[(curr_idx + 1) % len(matched)]
else:
    next_node = matched[0]

subprocess.run([msg_cmd, f'[con_id={next_node[\"id\"]}] focus'], check=False)
" "$key" "$cmd" "$pattern" "$match_mode" "$msg_cmd"
}
