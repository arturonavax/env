import Gio from 'gi://Gio';
import GLib from 'gi://GLib';
import Meta from 'gi://Meta';
import Shell from 'gi://Shell';
import * as Main from 'resource:///org/gnome/shell/ui/main.js';
import { Extension } from 'resource:///org/gnome/shell/extensions/extension.js';

const RCMD_IFACE_XML = `
<node>
  <interface name="dev.arturonavax.rcmd">
    <method name="Trigger">
      <arg type="s" name="key" direction="in"/>
      <arg type="s" name="command" direction="in"/>
      <arg type="s" name="pattern" direction="in"/>
      <arg type="s" name="mode" direction="in"/>
      <arg type="b" name="success" direction="out"/>
    </method>
  </interface>
  <interface name="org.gnome.Shell.Extensions.RunOrRaise">
    <method name="Call">
      <arg type="s" name="line" direction="in"/>
      <arg type="s" name="response" direction="out"/>
    </method>
  </interface>
  <interface name="cz.edvard.run_or_raise">
    <method name="Trigger">
      <arg type="s" name="target" direction="in"/>
      <arg type="b" name="success" direction="out"/>
    </method>
  </interface>
</node>
`;

function getCandidateNames(win) {
    const names = [];
    if (!win) return names;

    const wmClass = (typeof win.get_wm_class === 'function') ? (win.get_wm_class() || '') : '';
    const wmInstance = (typeof win.get_wm_class_instance === 'function') ? (win.get_wm_class_instance() || '') : '';
    const gtkAppId = (typeof win.get_gtk_application_id === 'function') ? (win.get_gtk_application_id() || '') : '';
    const sandboxedAppId = (typeof win.get_sandboxed_app_id === 'function') ? (win.get_sandboxed_app_id() || '') : '';

    for (const name of [wmClass, wmInstance, gtkAppId, sandboxedAppId]) {
        if (!name) continue;
        names.push(name);
        const parts = name.split('.');
        const lastPart = parts[parts.length - 1];
        if (lastPart && lastPart !== name) names.push(lastPart);

        const cleaned = name.replace(/^(gnome|xfce4|xfce|kde|google|org|com|io)-/i, '');
        if (cleaned && cleaned !== name) names.push(cleaned);

        const cleanedLast = lastPart.replace(/^(gnome|xfce4|xfce|kde|google|org|com|io)-/i, '');
        if (cleanedLast && cleanedLast !== lastPart) names.push(cleanedLast);
    }

    const pid = (typeof win.get_pid === 'function') ? win.get_pid() : 0;
    if (pid && pid > 0) {
        try {
            const [ok, comm] = GLib.file_get_contents(`/proc/${pid}/comm`);
            if (ok) {
                const commStr = new TextDecoder().decode(comm).trim();
                names.push(commStr);
                const cleanedComm = commStr.replace(/^(gnome|xfce4|xfce|kde)-/i, '')
                                           .replace(/-(bin|real|wrapped)$/i, '');
                if (cleanedComm !== commStr) names.push(cleanedComm);
            }
        } catch (_) {}
    }

    const title = (typeof win.get_title === 'function') ? (win.get_title() || '') : '';
    if (title) {
        const sepMatch = title.match(/[\s—\-]+([^\s—\-]+)$/);
        if (sepMatch && sepMatch[1]) {
            names.push(sepMatch[1]);
        }
        const firstWord = title.trim().split(/\s+/)[0];
        if (firstWord) names.push(firstWord);
    }

    return names;
}

function isMatch(win, pattern, mode) {
    if (!win || !pattern) return false;
    const patLower = pattern.toLowerCase();

    if (mode === 'title') {
        const title = (typeof win.get_title === 'function') ? (win.get_title() || '').toLowerCase() : '';
        return title.includes(patLower);
    }

    const wmClass = (typeof win.get_wm_class === 'function') ? (win.get_wm_class() || '').toLowerCase() : '';
    const wmInstance = (typeof win.get_wm_class_instance === 'function') ? (win.get_wm_class_instance() || '').toLowerCase() : '';
    const gtkAppId = (typeof win.get_gtk_application_id === 'function') ? (win.get_gtk_application_id() || '').toLowerCase() : '';
    const sandboxedAppId = (typeof win.get_sandboxed_app_id === 'function') ? (win.get_sandboxed_app_id() || '').toLowerCase() : '';

    if (wmClass.includes(patLower) || wmInstance.includes(patLower) || gtkAppId.includes(patLower) || sandboxedAppId.includes(patLower)) {
        return true;
    }

    const candidates = getCandidateNames(win);
    return candidates.some(cand => cand.toLowerCase().includes(patLower));
}

function launchApp(command) {
    if (!command) return;
    try {
        const appSys = Shell.AppSystem.get_default();
        const firstWord = command.trim().split(/\s+/)[0];
        const app = appSys.lookup_app(command) ||
                    appSys.lookup_app(`${command}.desktop`) ||
                    appSys.lookup_app(firstWord) ||
                    appSys.lookup_app(`${firstWord}.desktop`) ||
                    appSys.lookup_app(`com.mitchellh.${firstWord}.desktop`) ||
                    appSys.lookup_app(`org.gnome.${firstWord}.desktop`);
        if (app && (command.trim() === firstWord)) {
            app.activate();
            return;
        }
    } catch (_) {}

    try {
        // Run via bash -c to ensure user PATH (~/.local/bin) and argument parsing work
        Gio.Subprocess.new(['/usr/bin/env', 'bash', '-c', command], Gio.SubprocessFlags.NONE);
    } catch (e) {
        console.error(`[rcmd-shell] Failed to launch command "${command}": ${e.message}`);
    }
}

export default class RcmdExtension extends Extension {
    enable() {
        this._dbus = Gio.DBusExportedObject.wrapJSObject(RCMD_IFACE_XML, this);
        this._dbus.export(Gio.DBus.session, '/dev/arturonavax/rcmd');

        this._dbusRor = Gio.DBusExportedObject.wrapJSObject(RCMD_IFACE_XML, this);
        this._dbusRor.export(Gio.DBus.session, '/org/gnome/Shell/Extensions/RunOrRaise');

        this._dbusLegacy = Gio.DBusExportedObject.wrapJSObject(RCMD_IFACE_XML, this);
        this._dbusLegacy.export(Gio.DBus.session, '/cz/edvard/run_or_raise');

        this._nameOwnerId = Gio.bus_own_name(
            Gio.BusType.SESSION,
            'dev.arturonavax.rcmd',
            Gio.BusNameOwnerFlags.NONE,
            null, null, null
        );

        console.log('[rcmd-shell] Extension enabled and D-Bus interfaces exported.');
    }

    disable() {
        if (this._nameOwnerId) {
            Gio.bus_unown_name(this._nameOwnerId);
            this._nameOwnerId = 0;
        }
        if (this._dbus) {
            this._dbus.flush();
            this._dbus.unexport();
            this._dbus = null;
        }
        if (this._dbusRor) {
            this._dbusRor.flush();
            this._dbusRor.unexport();
            this._dbusRor = null;
        }
        if (this._dbusLegacy) {
            this._dbusLegacy.flush();
            this._dbusLegacy.unexport();
            this._dbusLegacy = null;
        }
        console.log('[rcmd-shell] Extension disabled.');
    }

    getWindows() {
        const tabListMode = (Meta.TabList && Meta.TabList.NORMAL_ALL !== undefined)
            ? Meta.TabList.NORMAL_ALL
            : 3;
        const windows = (global.display && typeof global.display.get_tab_list === 'function')
            ? global.display.get_tab_list(tabListMode, null)
            : [];
        return windows.filter(w => {
            if (!w || (typeof w.is_override_redirect === 'function' && w.is_override_redirect())) return false;
            const type = (typeof w.get_window_type === 'function') ? w.get_window_type() : Meta.WindowType.NORMAL;
            return type === Meta.WindowType.NORMAL || type === Meta.WindowType.DIALOG;
        });
    }

    Trigger(key, command, pattern, mode) {
        const windows = this.getWindows();
        const focused = (global.display && typeof global.display.get_focus_window === 'function')
            ? global.display.get_focus_window()
            : (global.display ? global.display.focus_window : null);

        // 1. Configured mode (command or pattern provided)
        if (pattern || command) {
            const pat = pattern || command;
            const matchMode = mode || 'class';
            const matching = windows.filter(w => isMatch(w, pat, matchMode));

            if (matching.length === 0) {
                if (command) {
                    launchApp(command);
                    return true;
                }
                return false;
            }

            const activeIndex = matching.findIndex(w => w === focused);
            if (activeIndex >= 0) {
                const nextIndex = (activeIndex + 1) % matching.length;
                Main.activateWindow(matching[nextIndex]);
            } else {
                // Coming from another app: raise sister windows behind target (macOS style)
                for (let i = matching.length - 1; i > 0; i--) {
                    try {
                        if (typeof matching[i].raise === 'function') {
                            matching[i].raise();
                        }
                    } catch (_) {}
                }
                Main.activateWindow(matching[0]);
            }
            return true;
        }

        // 2. Dynamic mode (unconfigured key letter: only open windows)
        const targetKey = (key || '').toLowerCase();
        if (!targetKey) return false;

        let targetAppWindows = [];
        for (const win of windows) {
            const candidates = getCandidateNames(win);
            const matches = candidates.some(cand => cand.toLowerCase().startsWith(targetKey));
            if (matches) {
                const wmClass = (typeof win.get_wm_class === 'function') ? (win.get_wm_class() || '') : '';
                const gtkAppId = (typeof win.get_gtk_application_id === 'function') ? (win.get_gtk_application_id() || '') : '';
                const appIdentifier = (wmClass || gtkAppId).toLowerCase();

                if (appIdentifier) {
                    targetAppWindows = windows.filter(w => {
                        const c = (typeof w.get_wm_class === 'function' ? (w.get_wm_class() || '') : '').toLowerCase();
                        const a = (typeof w.get_gtk_application_id === 'function' ? (w.get_gtk_application_id() || '') : '').toLowerCase();
                        return c === appIdentifier || a === appIdentifier;
                    });
                }
                if (targetAppWindows.length === 0) targetAppWindows = [win];
                break;
            }
        }

        if (targetAppWindows.length === 0) {
            return false;
        }

        const activeIndex = targetAppWindows.findIndex(w => w === focused);
        if (activeIndex >= 0) {
            const nextIndex = (activeIndex + 1) % targetAppWindows.length;
            Main.activateWindow(targetAppWindows[nextIndex]);
        } else {
            // Raise sister windows behind target
            for (let i = targetAppWindows.length - 1; i > 0; i--) {
                try {
                    if (typeof targetAppWindows[i].raise === 'function') {
                        targetAppWindows[i].raise();
                    }
                } catch (_) {}
            }
            Main.activateWindow(targetAppWindows[0]);
        }
        return true;
    }

    Call(line) {
        const parts = (line || '').split(',').map(s => s.trim());
        const cmd = parts[1] || '';
        const wmClass = parts[2] || '';
        const title = parts[3] || '';

        let pat = wmClass;
        let mode = 'class';
        if (title) {
            pat = title;
            mode = 'title';
        } else if (!pat) {
            pat = cmd;
        }

        this.Trigger('', cmd, pat, mode);
        return "Success";
    }
}
