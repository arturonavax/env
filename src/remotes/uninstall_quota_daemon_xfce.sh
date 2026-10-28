#!/usr/bin/env bash
# Compatibility uninstaller forwarding to aitools daemon
if command -v aitools >/dev/null 2>&1; then
	exec aitools daemon uninstall "$@"
fi
if command -v systemctl >/dev/null 2>&1; then
	systemctl --user stop quota-monitor.service quota-primer.timer quota-primer.service aitools-monitor.service 2>/dev/null || true
	systemctl --user disable quota-monitor.service quota-primer.timer quota-primer.service aitools-monitor.service 2>/dev/null || true
	systemctl --user daemon-reload 2>/dev/null || true
fi
