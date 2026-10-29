#!/usr/bin/env bash
# Compatibility uninstaller forwarding to aitools daemon and purging legacy daemon
if command -v aitools >/dev/null 2>&1; then
	aitools daemon uninstall "$@"
fi
if command -v systemctl >/dev/null 2>&1; then
	systemctl --user stop quota-monitor.service quota-primer.timer quota-primer.service aitools-monitor.service aitools-wake.timer aitools-wake.service 2>/dev/null || true
	systemctl --user disable quota-monitor.service quota-primer.timer quota-primer.service aitools-monitor.service aitools-wake.timer aitools-wake.service 2>/dev/null || true
	rm -f "$HOME/.config/systemd/user/quota-monitor.service" \
		"$HOME/.config/systemd/user/quota-primer.service" \
		"$HOME/.config/systemd/user/quota-primer.timer" 2>/dev/null || true
	systemctl --user daemon-reload 2>/dev/null || true
fi
rm -f "$HOME/.local/bin/quota-ctl" "/usr/local/bin/quota-ctl" 2>/dev/null || true
rm -rf "$HOME/.config/quota-primer" "$HOME/.local/state/quota_primer" 2>/dev/null || true
