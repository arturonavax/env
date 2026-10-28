#!/usr/bin/env bash
# ==============================================================================
# Quota Primer & Auto-Wake Daemon Installer / Manager for Linux & Xfce
# ==============================================================================
# Architecture:
#   1. Structured Telemetry: Real-time quota metrics via `agy -p /usage --output-format json`.
#   2. Intelligent Idempotency: Inspects remaining 5-hour rolling windows; only pings
#      models whose timer is expired or inactive, saving 100% of unused tokens.
#   3. Extensible Configuration: Fully declarative model schema in
#      `~/.config/quota-primer/config.json`. Add any model (Gemini, Claude, GPT) or CLI.
#   4. Hardware RTC Auto-Wake (04:00 AM) & Auto-Resuspend:
#      Wakes from S3 sleep, waits for Wi-Fi, primes expired models, schedules next day alarm,
#      and automatically returns to sleep in 12s to protect battery life.
#   5. Native User D-Bus Monitor: Catches `org.freedesktop.login1.PrepareForSleep`
#      with full user environment, credential store, and zero root requirement.
#   6. Clean Uninstallation: Integrated `--uninstall` and `--purge` flags.
# ==============================================================================

set -euo pipefail

if [ "${BASH_SOURCE[0]}" != "$0" ]; then
    echo "ERROR: Do not run this script with 'source' or '.'. Run: bash $0 or ./${0##*/}" >&2
    return 1 2>/dev/null || exit 1
fi

if [ "$EUID" -eq 0 ]; then
    echo "ERROR: Run this script as a normal user (it will prompt for sudo when necessary)." >&2
    exit 1
fi

BIN_DIR="$HOME/.local/bin"
CONFIG_DIR="$HOME/.config/quota-primer"
STATE_DIR="$HOME/.local/state/quota_primer"
SYSTEMD_USER_DIR="$HOME/.config/systemd/user"
DAEMON_SCRIPT="$BIN_DIR/quota-ctl"
CONFIG_FILE="$CONFIG_DIR/config.json"
SUDOERS_FILE="/etc/sudoers.d/quota-primer-rtcwake"
SERVICE_FILE="$SYSTEMD_USER_DIR/quota-monitor.service"
CURRENT_USER="$(id -un)"

show_help() {
    cat << EOF
Quota Primer & Auto-Wake Daemon Installer for Linux / Xfce

Usage:
  ./$(basename "$0") [OPTIONS]

Options:
  -i, --install     Install and enable quota-ctl daemon and systemd monitor (default)
  -u, --uninstall   Stop and uninstall quota-ctl daemon, services and sudoers rules
  -p, --purge       Uninstall and delete all user configuration and state directories
  -s, --status      Check current service and quota status
  -h, --help        Show this help message

File Locations:
  Binary:         $DAEMON_SCRIPT
  Configuration:  $CONFIG_FILE
  State:          $STATE_DIR/state.json
  Systemd Unit:   $SERVICE_FILE
  Sudoers Rule:   $SUDOERS_FILE (for passwordless rtcwake)

Examples:
  ./$(basename "$0") --install      # Standard installation
  ./$(basename "$0") --status       # Verify running status
  ./$(basename "$0") --uninstall    # Clean removal
EOF
}

uninstall_daemon() {
    local purge="${1:-false}"
    echo "==> Uninstalling Quota Primer Daemon..."

    # 1. Stop and disable systemd user units
    if command -v systemctl >/dev/null 2>&1; then
        echo "  • Stopping systemd user services..."
        systemctl --user stop quota-monitor.service quota-primer.timer quota-primer.service 2>/dev/null || true
        systemctl --user disable quota-monitor.service quota-primer.timer quota-primer.service 2>/dev/null || true
        rm -f "$SYSTEMD_USER_DIR/quota-monitor.service" \
              "$SYSTEMD_USER_DIR/quota-primer.service" \
              "$SYSTEMD_USER_DIR/quota-primer.timer"
        systemctl --user daemon-reload 2>/dev/null || true
    fi

    # 2. Disable active hardware RTC alarms
    if command -v rtcwake >/dev/null 2>&1; then
        echo "  • Disabling hardware RTC alarms..."
        sudo -n rtcwake -m disable 2>/dev/null || sudo rtcwake -m disable 2>/dev/null || true
    fi

    # 3. Remove sudoers rule
    if [ -f "$SUDOERS_FILE" ]; then
        echo "  • Removing sudoers rule: $SUDOERS_FILE..."
        sudo -n rm -f "$SUDOERS_FILE" 2>/dev/null || sudo rm -f "$SUDOERS_FILE" 2>/dev/null || true
    fi

    # 4. Remove binaries and desktop shortcuts
    echo "  • Removing executable: $DAEMON_SCRIPT..."
    rm -f "$DAEMON_SCRIPT"

    # 5. Purge configs if requested
    if [ "$purge" = "true" ]; then
        echo "  • Purging configuration ($CONFIG_DIR) and state ($STATE_DIR)..."
        rm -rf "$CONFIG_DIR" "$STATE_DIR"
    else
        echo "  • Preserving configuration ($CONFIG_DIR) and history ($STATE_DIR)."
        echo "    (Use --purge to remove configuration files completely)."
    fi

    echo "==> Quota Primer uninstallation complete."
}

install_daemon() {
    echo "==> [1/4] Preparing directories..."
    mkdir -p "$BIN_DIR" "$CONFIG_DIR" "$STATE_DIR" "$SYSTEMD_USER_DIR"

    echo "==> [2/4] Deploying configuration template ($CONFIG_FILE)..."
    if [ ! -f "$CONFIG_FILE" ]; then
cat << 'EOF' > "$CONFIG_FILE"
{
  "auto_wake": {
    "enabled": false,
    "time": "04:00",
    "auto_resuspend": true,
    "resuspend_delay_seconds": 12,
    "max_wake_window_minutes": 15
  },
  "active_hours": {
    "start": "03:55",
    "end": "20:00"
  },
  "targets": [
    {
      "id": "gemini",
      "name": "Gemini Models",
      "bucket_id": "gemini-5h",
      "enabled": true,
      "command": [
        "agy",
        "-p",
        "Reply with: 1",
        "--model",
        "gemini-3.8-flash-low",
        "--effort",
        "low"
      ],
      "timeout_sec": 20
    },
    {
      "id": "claude-gpt",
      "name": "Claude and GPT models",
      "bucket_id": "3p-5h",
      "enabled": true,
      "command": [
        "agy",
        "-p",
        "Reply with: 1",
        "--model",
        "claude-sonnet-5-5-low",
        "--effort",
        "low"
      ],
      "timeout_sec": 25
    },
    {
      "id": "gemini-argon",
      "name": "Gemini 4 Argon",
      "bucket_id": "gemini-argon-wtu-weekly",
      "enabled": false,
      "command": [
        "agy",
        "-p",
        "Reply with: 1",
        "--model",
        "gemini-4-argon-low",
        "--effort",
        "low"
      ],
      "timeout_sec": 25
    }
  ],
  "telegram": {
    "enabled": false,
    "bot_token": "",
    "chat_id": ""
  }
}
EOF
        echo "  • Created default configuration with Gemini and Claude targets."
    else
        echo "  • Preserved existing configuration in $CONFIG_FILE."
    fi

    echo "==> [3/4] Installing daemon executable ($DAEMON_SCRIPT)..."
cat << 'EOF' > "$DAEMON_SCRIPT"
#!/usr/bin/env python3
"""
quota-ctl: Intelligent AI Quota Primer & RTC Auto-Wake Manager for Linux & Xfce.
Supports Antigravity CLI (agy), Gemini, Claude, GPT, and arbitrary custom model commands.
"""
import json
import os
import re
import socket
import subprocess
import sys
import time
import urllib.parse
import urllib.request
from datetime import datetime, timedelta, timezone

CONFIG_DIR = os.path.expanduser("~/.config/quota-primer")
CONFIG_FILE = os.path.join(CONFIG_DIR, "config.json")
STATE_DIR = os.path.expanduser("~/.local/state/quota_primer")
STATE_FILE = os.path.join(STATE_DIR, "state.json")

# Terminal styling ANSI codes
C_RESET = "\033[0m"
C_BOLD = "\033[1m"
C_GREEN = "\033[32m"
C_YELLOW = "\033[33m"
C_BLUE = "\033[34m"
C_CYAN = "\033[36m"
C_RED = "\033[31m"
C_GRAY = "\033[90m"

DEFAULT_CONFIG = {
    "auto_wake": {
        "enabled": False,
        "time": "04:00",
        "auto_resuspend": True,
        "resuspend_delay_seconds": 12,
        "max_wake_window_minutes": 15
    },
    "active_hours": {
        "start": "03:55",
        "end": "20:00"
    },
    "targets": [
        {
            "id": "gemini",
            "name": "Gemini Models",
            "bucket_id": "gemini-5h",
            "enabled": True,
            "command": ["agy", "-p", "Reply with: 1", "--model", "gemini-3.8-flash-low", "--effort", "low"],
            "timeout_sec": 20
        },
        {
            "id": "claude-gpt",
            "name": "Claude and GPT models",
            "bucket_id": "3p-5h",
            "enabled": True,
            "command": ["agy", "-p", "Reply with: 1", "--model", "claude-sonnet-5-5-low", "--effort", "low"],
            "timeout_sec": 25
        },
        {
            "id": "gemini-argon",
            "name": "Gemini 4 Argon",
            "bucket_id": "gemini-argon-wtu-weekly",
            "enabled": False,
            "command": ["agy", "-p", "Reply with: 1", "--model", "gemini-4-argon-low", "--effort", "low"],
            "timeout_sec": 25
        }
    ],
    "telegram": {
        "enabled": False,
        "bot_token": "",
        "chat_id": ""
    }
}


def load_config() -> dict:
    os.makedirs(CONFIG_DIR, exist_ok=True)
    if not os.path.exists(CONFIG_FILE):
        with open(CONFIG_FILE, "w", encoding="utf-8") as f:
            json.dump(DEFAULT_CONFIG, f, indent=2)
        return DEFAULT_CONFIG
    try:
        with open(CONFIG_FILE, "r", encoding="utf-8") as f:
            data = json.load(f)
            merged = DEFAULT_CONFIG.copy()
            merged.update(data)
            return merged
    except Exception as e:
        print(f"[WARN] Error reading {CONFIG_FILE}: {e}", file=sys.stderr)
        return DEFAULT_CONFIG


def save_config(cfg: dict) -> None:
    os.makedirs(CONFIG_DIR, exist_ok=True)
    tmp_path = CONFIG_FILE + ".tmp"
    with open(tmp_path, "w", encoding="utf-8") as f:
        json.dump(cfg, f, indent=2)
    os.replace(tmp_path, CONFIG_FILE)


def load_state() -> dict:
    os.makedirs(STATE_DIR, exist_ok=True)
    default_state = {
        "scheduled_wake_epoch": 0,
        "expecting_auto_wake": False,
        "last_wake_epoch": 0,
        "history": []
    }
    if not os.path.exists(STATE_FILE):
        return default_state
    try:
        with open(STATE_FILE, "r", encoding="utf-8") as f:
            data = json.load(f)
            default_state.update(data)
            return default_state
    except Exception:
        return default_state


def save_state(st: dict) -> None:
    os.makedirs(STATE_DIR, exist_ok=True)
    tmp_path = STATE_FILE + ".tmp"
    with open(tmp_path, "w", encoding="utf-8") as f:
        json.dump(st, f, indent=2)
    os.replace(tmp_path, STATE_FILE)


def wait_for_internet(timeout_sec: int = 30) -> bool:
    deadline = time.time() + timeout_sec
    while time.time() < deadline:
        for host, port in [("1.1.1.1", 443), ("8.8.8.8", 53)]:
            try:
                with socket.create_connection((host, port), timeout=2):
                    return True
            except OSError:
                pass
        time.sleep(1.2)
    return False


def send_notification(title: str, body: str, urgency: str = "normal") -> None:
    env = os.environ.copy()
    uid = os.getuid()
    env.setdefault("DISPLAY", ":0")
    env.setdefault("DBUS_SESSION_BUS_ADDRESS", f"unix:path=/run/user/{uid}/bus")
    try:
        subprocess.run(
            ["notify-send", "-u", urgency, "-a", "QuotaPrimer", "-i", "utilities-terminal", title, body],
            env=env,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            check=False,
        )
    except FileNotFoundError:
        pass

    cfg = load_config()
    tg_cfg = cfg.get("telegram", {})
    tg_token = tg_cfg.get("bot_token") or os.getenv("TELEGRAM_BOT_TOKEN", "")
    tg_chat = tg_cfg.get("chat_id") or os.getenv("TELEGRAM_CHAT_ID", "")

    if tg_token and tg_chat:
        url = f"https://api.telegram.org/bot{tg_token}/sendMessage"
        payload = urllib.parse.urlencode({
            "chat_id": tg_chat,
            "text": f"⚡ *{title}*\n{body}",
            "parse_mode": "Markdown"
        }).encode("utf-8")
        try:
            urllib.request.urlopen(urllib.request.Request(url, data=payload), timeout=5)
        except Exception:
            pass


def fetch_agy_usage() -> dict:
    """Executes 'agy -p /usage --output-format json' and returns command data."""
    try:
        res = subprocess.run(
            ["agy", "-p", "/usage", "--output-format", "json"],
            capture_output=True,
            text=True,
            timeout=25
        )
        if res.returncode == 0 and res.stdout.strip():
            parsed = json.loads(res.stdout)
            cmd_data = parsed.get("command", {}).get("data", {})
            if cmd_data:
                return cmd_data
    except Exception as e:
        print(f"[WARN] Error fetching /usage from agy: {e}", file=sys.stderr)

    # Fallback text parsing
    try:
        res = subprocess.run(["agy", "-p", "/usage"], capture_output=True, text=True, timeout=20)
        if res.returncode == 0 and res.stdout.strip():
            return parse_text_usage(res.stdout)
    except Exception:
        pass

    return {}


def parse_text_usage(text: str) -> dict:
    groups = []
    lines = text.strip().splitlines()
    for line in lines:
        parts = re.split(r'\s{2,}|\t', line.strip())
        if len(parts) >= 3:
            g_name, b_name, rem_pct = parts[0], parts[1], parts[2]
            reset_str = parts[3] if len(parts) > 3 else ""
            b_id = "gemini-5h" if "Five" in b_name and "Gemini" in g_name else "3p-5h"
            rem_frac = float(rem_pct.replace("%", "").strip()) / 100.0 if "%" in rem_pct else 1.0
            groups.append({
                "name": g_name,
                "buckets": [{
                    "id": b_id,
                    "name": b_name,
                    "remaining_fraction": rem_frac,
                    "reset_time": reset_str
                }]
            })
    return {"groups": groups}


def evaluate_targets(config: dict, usage_data: dict) -> list[dict]:
    now_utc = datetime.now(timezone.utc)
    evaluated = []

    bucket_map = {}
    for group in usage_data.get("groups", []):
        for bucket in group.get("buckets", []):
            b_id = bucket.get("id")
            if b_id:
                bucket_map[b_id] = bucket

    for target in config.get("targets", []):
        t_id = target.get("id")
        t_name = target.get("name", t_id)
        bucket_id = target.get("bucket_id")
        bucket = bucket_map.get(bucket_id)

        info = {
            "id": t_id,
            "name": t_name,
            "bucket_id": bucket_id,
            "enabled": target.get("enabled", True),
            "command": target.get("command", []),
            "timeout_sec": target.get("timeout_sec", 25),
            "remaining_fraction": 1.0,
            "remaining_percent": 100.0,
            "reset_time_iso": None,
            "reset_time_local": None,
            "is_active": False,
            "remaining_seconds": 0,
            "status_label": "IDLE (no active window)",
            "raw_bucket": bucket
        }

        if bucket:
            r_frac = float(bucket.get("remaining_fraction", 1.0))
            r_time_str = bucket.get("reset_time")
            info["remaining_fraction"] = r_frac
            info["remaining_percent"] = round(r_frac * 100.0, 1)
            info["reset_time_iso"] = r_time_str

            if r_time_str:
                try:
                    if "T" in r_time_str and "Z" in r_time_str:
                        r_dt = datetime.fromisoformat(r_time_str.replace("Z", "+00:00"))
                    else:
                        clean_str = re.sub(r'\s*-\d{2}$', '', r_time_str.strip())
                        r_dt = datetime.strptime(clean_str, "%Y-%m-%d %H:%M").replace(tzinfo=timezone.utc)

                    local_dt = r_dt.astimezone()
                    info["reset_time_local"] = local_dt.strftime("%H:%M:%S")

                    if r_dt > now_utc:
                        info["is_active"] = True
                        diff_sec = (r_dt - now_utc).total_seconds()
                        info["remaining_seconds"] = int(diff_sec)
                        h = int(diff_sec // 3600)
                        m = int((diff_sec % 3600) // 60)
                        info["status_label"] = f"ACTIVE ({h}h {m:02d}m remaining, resets at {local_dt.strftime('%H:%M')})"
                    else:
                        info["is_active"] = False
                        info["status_label"] = f"EXPIRED (100% refreshed since {local_dt.strftime('%H:%M')})"
                except Exception:
                    info["status_label"] = f"INVALID_DATE ({r_time_str})"
            else:
                info["status_label"] = "IDLE (no time window)"
        else:
            info["status_label"] = "NO_BUCKET_DATA"

        evaluated.append(info)

    return evaluated


def fire_target_wakeup(target: dict) -> tuple[bool, str]:
    cmd = target.get("command", [])
    if not cmd:
        return False, "Command not configured"
    timeout = target.get("timeout_sec", 25)
    t0 = time.time()
    try:
        proc = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=timeout
        )
        elapsed = round(time.time() - t0, 2)
        if proc.returncode == 0:
            out_sample = proc.stdout.strip().replace("\n", " ")[:50] or "ok"
            return True, f"Success in {elapsed}s (Response: '{out_sample}')"
        else:
            err = proc.stderr.strip()[:100] or proc.stdout.strip()[:100] or f"Exit {proc.returncode}"
            return False, f"Failed in {elapsed}s: {err}"
    except subprocess.TimeoutExpired:
        return False, f"Timeout ({timeout}s)"
    except Exception as e:
        return False, f"Exception: {e}"


def prime_quotas(force: bool = False, target_id_filter: str = None, reason: str = "manual") -> dict:
    config = load_config()
    state = load_state()

    print(f"{C_BOLD}{C_CYAN}==> Verifying internet connection...{C_RESET}")
    if not wait_for_internet(timeout_sec=25):
        msg = "No internet connection detected. Quota wake-up aborted."
        print(f"{C_RED}[ERROR] {msg}{C_RESET}", file=sys.stderr)
        send_notification("⚠️ Quota Primer: No Network", msg, urgency="critical")
        return {"success": False, "error": "no_internet"}

    print(f"{C_BOLD}{C_CYAN}==> Fetching live quota telemetry from agy (/usage)...{C_RESET}")
    usage_data = fetch_agy_usage()
    targets = evaluate_targets(config, usage_data)

    results = {}
    woken = []
    skipped = []

    for t in targets:
        t_id = t["id"]
        if target_id_filter and t_id != target_id_filter:
            continue
        if not t["enabled"] and not force:
            continue

        print(f"  • {C_BOLD}{t['name']}{C_RESET} ({t_id}):")
        print(f"    - Current status: {t['status_label']}")
        print(f"    - Remaining: {t['remaining_percent']}%")

        if force or not t["is_active"]:
            print(f"    {C_YELLOW}⚡ Waking up model (sending 1-token trigger prompt)...{C_RESET}")
            ok, msg = fire_target_wakeup(t)
            results[t_id] = {"primed": True, "ok": ok, "message": msg}
            if ok:
                print(f"    {C_GREEN}✓ {msg}{C_RESET}")
                woken.append(f"{t['name']}: {msg}")
            else:
                print(f"    {C_RED}✗ {msg}{C_RESET}")
                woken.append(f"{t['name']} [FAILED]: {msg}")
        else:
            print(f"    {C_GREEN}✓ Timer already running ({t['status_label']}). Skipping to preserve quota.{C_RESET}")
            results[t_id] = {"primed": False, "ok": True, "message": t['status_label']}
            skipped.append(f"{t['name']}: {t['status_label']}")

    now_local = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    hist_entry = {
        "timestamp": now_local,
        "reason": reason,
        "results": results
    }
    state.setdefault("history", []).append(hist_entry)
    state["history"] = state["history"][-50:]
    state["last_wake_epoch"] = time.time()
    save_state(state)

    lines = []
    if woken:
        lines.append("⚡ Models Activated:")
        for w in woken:
            lines.append(f"  • {w}")
    if skipped:
        lines.append("⏳ Already Active:")
        for s in skipped:
            lines.append(f"  • {s}")

    send_notification(
        f"⚡ Quota Primer ({reason})",
        "\n".join(lines) if lines else "All quotas verified.",
        urgency="normal"
    )

    return {"success": True, "woken": woken, "skipped": skipped, "results": results}


def next_wake_epoch(time_str: str) -> int:
    h, m = map(int, time_str.split(":"))
    now = datetime.now()
    target = now.replace(hour=h, minute=m, second=0, microsecond=0)
    if now >= target:
        target += timedelta(days=1)
    return int(target.timestamp())


def arm_rtc_alarm() -> bool:
    config = load_config()
    auto_wake = config.get("auto_wake", {})
    if not auto_wake.get("enabled", False):
        print(f"{C_YELLOW}[INFO] Auto-Wake disabled in configuration.{C_RESET}")
        return False

    target_time = auto_wake.get("time", "04:00")
    target_epoch = next_wake_epoch(target_time)
    target_dt = datetime.fromtimestamp(target_epoch)

    state = load_state()
    state["scheduled_wake_epoch"] = target_epoch
    state["expecting_auto_wake"] = True
    save_state(state)

    res = subprocess.run(["sudo", "-n", "/usr/sbin/rtcwake", "-m", "no", "-t", str(target_epoch)], check=False)
    if res.returncode == 0:
        print(f"{C_GREEN}✓ Hardware RTC alarm programmed for: {target_dt.strftime('%Y-%m-%d %H:%M:%S %Z')}{C_RESET}")
        return True
    else:
        print(f"{C_RED}✗ Failed to execute rtcwake (is sudoers rule configured?){C_RESET}", file=sys.stderr)
        return False


def disable_rtc_alarm() -> bool:
    state = load_state()
    state["expecting_auto_wake"] = False
    save_state(state)
    res = subprocess.run(["sudo", "-n", "/usr/sbin/rtcwake", "-m", "disable"], check=False)
    return res.returncode == 0


def handle_on_suspend():
    config = load_config()
    auto_wake = config.get("auto_wake", {})
    if auto_wake.get("enabled", False):
        arm_rtc_alarm()


def handle_on_resume():
    state = load_state()
    config = load_config()
    now_epoch = time.time()
    scheduled_epoch = state.get("scheduled_wake_epoch", 0)
    expecting = state.get("expecting_auto_wake", False)
    auto_wake_cfg = config.get("auto_wake", {})
    max_window_sec = auto_wake_cfg.get("max_wake_window_minutes", 15) * 60

    is_rtc_wake = expecting and abs(now_epoch - scheduled_epoch) < max_window_sec

    if is_rtc_wake:
        print(f"{C_BOLD}{C_CYAN}==> Detected scheduled RTC Auto-Wake. Initiating activation cycle...{C_RESET}")
        state["expecting_auto_wake"] = False
        save_state(state)

        # 1. Wake quotas
        prime_quotas(force=False, reason="rtc_04am_auto_wake")

        # 2. Schedule next day's alarm
        arm_rtc_alarm()

        # 3. Auto-resuspend
        if auto_wake_cfg.get("auto_resuspend", True):
            delay = auto_wake_cfg.get("resuspend_delay_seconds", 12)
            print(f"{C_YELLOW}💤 Auto-resuspending in {delay} seconds to preserve battery...{C_RESET}")
            send_notification(
                "💤 Auto-Resuspending System",
                f"Model quotas successfully primed. Suspending in {delay}s...",
                urgency="normal"
            )
            time.sleep(delay)
            subprocess.run(["systemctl", "suspend"], check=False)
    else:
        print(f"{C_CYAN}==> Manual user resume detected.{C_RESET}")
        wait_for_internet(timeout_sec=15)
        usage_data = fetch_agy_usage()
        targets = evaluate_targets(config, usage_data)
        lines = []
        for t in targets:
            if t["enabled"]:
                lines.append(f"• {t['name']}: {t['status_label']}")
        if lines:
            send_notification(
                "💻 System Resumed — Quota Status",
                "\n".join(lines),
                urgency="low"
            )


def print_status_table():
    config = load_config()
    state = load_state()
    usage_data = fetch_agy_usage()
    targets = evaluate_targets(config, usage_data)

    print(f"\n{C_BOLD}{C_CYAN}=============================================================================={C_RESET}")
    print(f"{C_BOLD}   QUOTA PRIMER & RTC AUTO-WAKE DASHBOARD (Antigravity & AI CLI){C_RESET}")
    print(f"{C_BOLD}{C_CYAN}=============================================================================={C_RESET}\n")

    auto_wake_cfg = config.get("auto_wake", {})
    auto_wake_on = auto_wake_cfg.get("enabled", False)
    wake_time = auto_wake_cfg.get("time", "04:00")
    auto_resuspend = auto_wake_cfg.get("auto_resuspend", True)

    wake_status = f"{C_GREEN}ENABLED (at {wake_time}){C_RESET}" if auto_wake_on else f"{C_GRAY}DISABLED{C_RESET}"
    resuspend_status = f"{C_GREEN}YES (sleeps after wake){C_RESET}" if auto_resuspend else f"{C_YELLOW}NO (stays awake){C_RESET}"

    print(f"  • {C_BOLD}RTC Hardware Auto-Wake{C_RESET}     : {wake_status}")
    print(f"  • {C_BOLD}Auto-Resuspend Mode{C_RESET}        : {resuspend_status}")
    print(f"  • {C_BOLD}Configuration File{C_RESET}         : {CONFIG_FILE}")

    rtc_out = subprocess.run(["sudo", "-n", "/usr/sbin/rtcwake", "-m", "show"], capture_output=True, text=True, check=False)
    if rtc_out.returncode == 0:
        alarm_line = [l.strip() for l in rtc_out.stdout.splitlines() if "alarm:" in l]
        if alarm_line:
            print(f"  • {C_BOLD}Active Hardware Alarm{C_RESET}      : {alarm_line[0]}")

    print(f"\n{C_BOLD}CONFIGURED MODELS & TARGETS:{C_RESET}")
    print(f"{'ID':<15} {'Name':<25} {'Enabled':<10} {'Quota':<10} {'Status / Reset Time'}")
    print(f"{'-'*15} {'-'*25} {'-'*10} {'-'*10} {'-'*30}")

    for t in targets:
        en_str = f"{C_GREEN}Yes{C_RESET}" if t["enabled"] else f"{C_GRAY}No{C_RESET}"
        rem_str = f"{t['remaining_percent']}%"
        if t["remaining_percent"] > 80:
            rem_str = f"{C_GREEN}{rem_str}{C_RESET}"
        elif t["remaining_percent"] > 30:
            rem_str = f"{C_YELLOW}{rem_str}{C_RESET}"
        else:
            rem_str = f"{C_RED}{rem_str}{C_RESET}"

        status_colored = t['status_label']
        if "ACTIVE" in status_colored:
            status_colored = f"{C_GREEN}{status_colored}{C_RESET}"
        elif "EXPIRED" in status_colored:
            status_colored = f"{C_YELLOW}{status_colored}{C_RESET}"

        print(f"{t['id']:<15} {t['name']:<25} {en_str:<19} {rem_str:<19} {status_colored}")

    print(f"\n{C_GRAY}Run 'quota-ctl wake' to prime models with inactive windows.{C_RESET}")
    print(f"{C_GRAY}Run 'quota-ctl auto-wake on/off' to toggle nightly hardware wake-up.{C_RESET}\n")


def print_genmon_panel():
    config = load_config()
    usage_data = fetch_agy_usage()
    targets = evaluate_targets(config, usage_data)

    auto_wake_cfg = config.get("auto_wake", {})
    auto_wake_on = "4AM:ON" if auto_wake_cfg.get("enabled", False) else "4AM:OFF"

    active_items = [t for t in targets if t["enabled"] and t["is_active"]]

    if not active_items:
        print(f"<txt>⚪ Quota: 100% [{auto_wake_on}]</txt>")
        print("<tool>All model quota windows are idle/available (100%).</tool>")
        return

    min_target = min(active_items, key=lambda x: x["remaining_seconds"])
    rem_sec = min_target["remaining_seconds"]
    h = rem_sec // 3600
    m = (rem_sec % 3600) // 60

    icon = "🔥" if rem_sec < 2700 else "⚡"
    print(f"<txt>{icon} {min_target['name'][:6]}: {h}h{m:02d}m [{auto_wake_on}]</txt>")

    tool_lines = []
    for t in targets:
        if t["enabled"]:
            tool_lines.append(f"{t['name']}: {t['remaining_percent']}% ({t['status_label']})")
    tool_lines.append(f"Auto-Wake: {auto_wake_on}")
    print(f"<tool>{' &#10; '.join(tool_lines)}</tool>")


def run_monitor_daemon():
    print(f"{C_BOLD}{C_CYAN}==> Starting D-Bus sleep & resume monitor daemon...{C_RESET}")
    proc = subprocess.Popen(
        ["gdbus", "monitor", "--system", "--dest", "org.freedesktop.login1"],
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        text=True
    )
    last_tick = time.time()
    try:
        while True:
            line = proc.stdout.readline()
            if not line:
                break
            if "PrepareForSleep: <true>" in line:
                print(f"{C_YELLOW}[D-Bus] Pre-sleep event detected (suspending system)...{C_RESET}")
                handle_on_suspend()
            elif "PrepareForSleep: <false>" in line:
                print(f"{C_GREEN}[D-Bus] Post-sleep event detected (resuming system)...{C_RESET}")
                time.sleep(2)
                handle_on_resume()

            if time.time() - last_tick > 60:
                last_tick = time.time()
    except KeyboardInterrupt:
        proc.terminate()


def print_cli_help():
    print(f"""{C_BOLD}quota-ctl: Intelligent AI Quota Primer & RTC Auto-Wake Manager{C_RESET}

{C_BOLD}USAGE:{C_RESET}
  quota-ctl <command> [options]

{C_BOLD}COMMANDS:{C_RESET}
  {C_CYAN}status{C_RESET}                  Show status dashboard of all models, percentages and timers
  {C_CYAN}usage{C_RESET}                   Display live structured JSON telemetry directly from agy /usage
  {C_CYAN}wake [target_id]{C_RESET}        Intelligently prime models whose 5h timer has expired or is idle
  {C_CYAN}force-wake [target_id]{C_RESET}  Force an activation ping unconditionally regardless of active timer
  {C_CYAN}auto-wake [on|off]{C_RESET}      Toggle automatic 04:00 AM RTC hardware alarm with re-suspension
  {C_CYAN}arm-wake{C_RESET}                Program hardware RTC alarm for next scheduled wake window
  {C_CYAN}panel{C_RESET}                   Emit XML formatted text for xfce4-genmon-plugin top panel widget
  {C_CYAN}monitor{C_RESET}                 Run background D-Bus sleep/resume listener (used by systemd unit)
  {C_CYAN}config{C_RESET}                  Show path and details of ~/.config/quota-primer/config.json
  {C_CYAN}help, --help, -h{C_RESET}        Display this help information

{C_BOLD}CONFIGURATION & EXTENSION:{C_RESET}
  Edit {C_CYAN}{CONFIG_FILE}{C_RESET} to customize wake times, enable/disable models, or add
  custom AI commands. Any target with an agy bucket_id will automatically track live quota.
""")


def main():
    if len(sys.argv) < 2 or sys.argv[1] in ("help", "--help", "-h"):
        print_cli_help()
        sys.exit(0)

    cmd = sys.argv[1]

    if cmd in ("status", "info"):
        print_status_table()
    elif cmd in ("usage", "telemetry"):
        usage_data = fetch_agy_usage()
        print(json.dumps(usage_data, indent=2))
    elif cmd in ("wake", "ping"):
        target_id = sys.argv[2] if len(sys.argv) > 2 else None
        prime_quotas(force=False, target_id_filter=target_id, reason="manual_wake")
    elif cmd in ("force-wake", "force-ping"):
        target_id = sys.argv[2] if len(sys.argv) > 2 else None
        prime_quotas(force=True, target_id_filter=target_id, reason="forced_wake")
    elif cmd in ("auto-wake", "auto-4am"):
        sub = sys.argv[2] if len(sys.argv) > 2 else "status"
        cfg = load_config()
        if sub in ("on", "true", "1", "enable"):
            cfg["auto_wake"]["enabled"] = True
            save_config(cfg)
            arm_rtc_alarm()
            print(f"{C_GREEN}✅ Auto-Wake 04:00 AM ENABLED with automatic re-suspension.{C_RESET}")
        elif sub in ("off", "false", "0", "disable"):
            cfg["auto_wake"]["enabled"] = False
            save_config(cfg)
            disable_rtc_alarm()
            print(f"{C_YELLOW}🛑 Auto-Wake 04:00 AM DISABLED.{C_RESET}")
        else:
            enabled = cfg.get("auto_wake", {}).get("enabled", False)
            print(f"Auto-Wake 04:00 AM: {C_GREEN + 'ON' if enabled else C_GRAY + 'OFF'}{C_RESET}")
    elif cmd == "arm-wake":
        arm_rtc_alarm()
    elif cmd == "on-suspend":
        handle_on_suspend()
    elif cmd == "on-resume":
        handle_on_resume()
    elif cmd == "monitor":
        run_monitor_daemon()
    elif cmd == "panel":
        print_genmon_panel()
    elif cmd in ("config", "edit"):
        print(f"Configuration file: {CONFIG_FILE}")
    else:
        print(f"Unknown command: '{cmd}'. Run 'quota-ctl --help' for available commands.")
        sys.exit(1)


if __name__ == "__main__":
    main()
EOF
    chmod +x "$DAEMON_SCRIPT"

    echo "==> [4/4] Configuring systemd user service ($SERVICE_FILE)..."
cat << EOF > "$SERVICE_FILE"
[Unit]
Description=Quota Primer D-Bus Sleep & Resume Monitor (Antigravity & AI Quotas)
After=graphical-session.target network.target

[Service]
Type=simple
ExecStart=$DAEMON_SCRIPT monitor
Restart=always
RestartSec=5
Environment=PATH=$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin
Environment=DISPLAY=:0
Environment=XAUTHORITY=$HOME/.Xauthority

[Install]
WantedBy=default.target graphical-session.target
EOF

    systemctl --user daemon-reload || true
    systemctl --user enable --now quota-monitor.service || true

    # Configure passwordless sudoers for rtcwake
    if [ -w "/etc/sudoers.d" ] || sudo -n true 2>/dev/null; then
        echo "  • Configuring passwordless rtcwake in $SUDOERS_FILE..."
        sudo tee "$SUDOERS_FILE" > /dev/null << EOS
$CURRENT_USER ALL=(root) NOPASSWD: /usr/sbin/rtcwake
EOS
        sudo chmod 0440 "$SUDOERS_FILE"
    fi

    echo ""
    echo "=============================================================================="
    echo "✅ Quota Primer & Auto-Wake Daemon installed successfully at: $DAEMON_SCRIPT"
    echo "=============================================================================="
    echo "Commands:"
    echo "  • quota-ctl status            -> View dashboard with remaining quota and reset times"
    echo "  • quota-ctl wake              -> Intelligently prime models with expired windows"
    echo "  • quota-ctl auto-wake on/off  -> Toggle automatic 04:00 AM hardware wake-up"
    echo "  • quota-ctl usage             -> Output raw JSON telemetry from agy /usage"
    echo "  • ./$(basename "$0") --uninstall -> Cleanly uninstall services and binaries"
    echo "=============================================================================="
}

ACTION="install"

while [[ $# -gt 0 ]]; do
    case "$1" in
        -i|--install)
            ACTION="install"
            shift
            ;;
        -u|--uninstall)
            ACTION="uninstall"
            shift
            ;;
        -p|--purge)
            ACTION="purge"
            shift
            ;;
        -s|--status)
            ACTION="status"
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            echo "Unknown option: $1" >&2
            show_help
            exit 1
            ;;
    esac
done

case "$ACTION" in
    install)
        install_daemon
        ;;
    uninstall)
        uninstall_daemon "false"
        ;;
    purge)
        uninstall_daemon "true"
        ;;
    status)
        if [ -x "$DAEMON_SCRIPT" ]; then
            "$DAEMON_SCRIPT" status
        else
            echo "Quota Primer is not installed at $DAEMON_SCRIPT"
        fi
        ;;
esac
