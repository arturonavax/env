#!/usr/bin/env bash
# ==============================================================================
# rcmd backend: Sway / i3 (wlroots / Wayland)
# Uses swaymsg to query and focus containers with dynamic matching.
# ==============================================================================

rcmd_backend_sway() {
    local key="$1"
    local cmd="$2"
    local pattern="$3"

    local msg_cmd="swaymsg"
    command -v swaymsg >/dev/null 2>&1 || msg_cmd="i3-msg"

    if ! command -v "$msg_cmd" >/dev/null 2>&1; then
        [ -n "$cmd" ] && nohup bash -c "$cmd" >/dev/null 2>&1 &
        return 0
    fi

    python3 -c "
import json, subprocess, sys

key = sys.argv[1].lower()
cmd = sys.argv[2]
pattern = sys.argv[3].lower()
msg_cmd = sys.argv[4]

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
    # Window leaf
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
    matched = [n for n in nodes if pattern in n['app_id'] or pattern in n['class'] or pattern in n['name']]
elif key:
    # Dynamic fallback
    target_cls = ''
    for n in nodes:
        if n['app_id'].startswith(key) or n['class'].startswith(key) or n['name'].startswith(key):
            target_cls = n['app_id'] or n['class']
            break
    if target_cls:
        matched = [n for n in nodes if n['app_id'] == target_cls or n['class'] == target_cls]

if not matched:
    if cmd:
        subprocess.Popen(f'{msg_cmd} exec \"{cmd}\"', shell=True)
    sys.exit(0)

# Check cycling
focused_in_matched = [n for n in matched if n['focused']]
if focused_in_matched:
    curr_idx = matched.index(focused_in_matched[0])
    next_node = matched[(curr_idx + 1) % len(matched)]
else:
    next_node = matched[0]

subprocess.run([msg_cmd, f'[con_id={next_node[\"id\"]}] focus'], check=False)
" "$key" "$cmd" "$pattern" "$msg_cmd"
}
