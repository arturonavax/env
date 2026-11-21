# Hardware & Desktop Optimization Guide: Dell Precision M6800

Comprehensive engineering reference and implementation guide for resolving Nouveau driver quirks, multi-monitor geometry desync, compositor leaks, browser rendering artifacts, and ACPI sleep disruptions on the **Dell Precision M6800** workstation.

---

## 1. Hardware & Software Topology

| Subsystem | Specification | Operational Constraint |
| :--- | :--- | :--- |
| **Host Machine** | Dell Precision M6800 (Workstation) | Dual-output dock / DisplayPort setup |
| **CPU** | Intel Core i7-4910MQ (8 threads @ 3.90 GHz) | Haswell microarchitecture (AVX2 supported, no hardware AV1/VP9 decode) |
| **Discrete GPU** | NVIDIA Quadro K3100M (GK104GLM Kepler, 4GB VRAM) | Proprietary NVIDIA 390.xx driver unsupported on modern Linux kernels (7.x) |
| **Kernel Driver** | `nouveau` (Open-source DRM/KMS) | Runs at minimum boot clock (135 MHz); lack of stable re-clocking & fragile DRI3 fence sync |
| **Displays** | **DP-2**: LG 27" 4K (3840x2160 @ 60Hz)<br>**eDP-1**: Internal 17" 1080p (1920x1080 @ 60Hz) | Total combined X11 root canvas: **5760 x 2160** pixels |
| **Desktop Environment** | Xfce 4.20 (`xfwm4` X11 window manager) | Native multi-head desktop with top panel (`xfce4-panel`) |
| **Compositor** | `picom` v12.5 (GLX backend) | Standalone composite manager handling tearing prevention |

---

## 2. Issues Diagnosed & Root Cause Analysis

### Issue 1: Ghostty + Herdr Progressive Input Latency & FPS Degradation
* **Symptom**: Ghostty typing latency increased steadily over hours, and frame rates progressively degraded during long sessions, especially with multiplexers (Herdr) and AI CLI agents streaming output. Closing and reopening the window temporarily cleared the lag.
* **Root Cause**:
  1. **AT-SPI Accessibility Leak (`gail:atk-bridge`)**: GTK4 by default loads the accessibility bridge module. For every terminal cell change, scroll event, and cursor movement, `atk-bridge` serializes accessible objects and emits D-Bus messages. In long-running sessions with heavy output, the GTK main thread event loop (responsible for processing keyboard input) spent over 90% of its CPU time processing accessibility tree churn (accumulating >70 minutes of CPU time).
  2. **PageList Scrollback Inflation**: By default, Ghostty allows up to 10,000,000 lines of scrollback. Under continuous agent streaming, Ghostty's internal `PageList` mmap memory pool continuously grows, causing dirty-page render traversal times to climb steadily.
  3. **32-Bit ARGB Visual**: `background-opacity < 1.0` forced 32-bit alpha-blending across 3840x2160 on every frame.
  4. **VSync Frame Queue Delay**: `window-vsync = true` forced keystrokes to wait for vertical display refresh (up to 16.6ms input delay).
* **Resolution**:
  - `~/.local/bin/ghostty`: High-performance wrapper executing with `GTK_A11Y=none`, `NO_AT_BRIDGE=1`, `GTK_MODULES=""`, and `GSK_RENDERER=gl`. Completely eliminates the AT-SPI accessibility CPU leak on the main thread.
  - `scrollback-limit = 10000` & `image-storage-limit = 50000000`: Bounded scrollback memory pool preventing `PageList` inflation while Herdr manages session history.
  - `background-opacity = 1.0`: Native 24-bit TrueColor opaque window.
  - `window-vsync = false`: Direct zero-latency keyboard input response.

### Issue 2: Brave Orange Shadows, Paste/Context-Menu Hangs, 15 FPS Video & Matrix Canvas GUI Crashes
* **Symptom**: Drop shadows rendered with an unnatural orange hue; `Ctrl-V` and right-click menus froze tabs; videos stuttered at ~15 FPS; and opening `arturonavax.dev` (with animated Matrix canvas + backdrop blur) completely froze the entire X11 graphical interface across both screens for 10 seconds until the GPU channel reset.
* **Root Cause**:
  1. **Orange Shadows**: Mesa Nouveau's Kepler driver has known shader math and texture format mismatches with `GL_ARB_shadow` when Chromium executes GPU-accelerated Skia rasterization (`Skia/Ganesh`).
  2. **Paste & Right-Click Lockups**: During paste or context menu generation, Chromium synchronously queries X11 clipboard formats (`TARGETS`). When the GPU process stalled in a Mesa fence wait rasterizing 4K surfaces, the X11 IPC thread blocked. "Force-paste" bypassed this by injecting text directly into the DOM node.
  3. **15 FPS Video Playback**: An initial attempt using `--use-gl=desktop` failed because Chromium deprecated that switch, causing fallback to `--use-gl=disabled`. In addition, `--no-xshm` disabled X11 shared memory (`MIT-SHM`), forcing raw 4K video frames over Unix domain sockets.
  4. **Matrix Canvas GUI Freeze**: Sites like `arturonavax.dev` render high-frequency 2D canvas streams (`OffscreenCanvas` / 2D context) underneath elements with CSS `backdrop-filter: blur(8px)`. When `--ignore-gpu-blocklist` and `--enable-zero-copy` were forced, Chromium attempted hardware-accelerated 2D canvas texture sampling across zero-copy buffers. Nouveau Kepler hit a kernel FIFO ringbuffer lockup, freezing the X11 server across both displays until the kernel DRM watchdog triggered a GPU reset (`Using 39-bit DMA addresses`).
* **Resolution**:
  - `--disable-accelerated-2d-canvas`: Offloads 2D canvas operations (such as the Matrix stream) to the 8-thread Core i7 CPU, completely preventing the GPU hardware lockup.
  - `--disable-gpu-rasterization` & `--canvas-oop-rasterization=false`: Eliminates orange shadows and shader compilation stalls.
  - `--enable-gpu-compositing`: Keeps GPU compositing active over standard X11 shared memory (MIT-SHM) for smooth 60 FPS video and UI presentation without zero-copy DMA crashes.
  - Removed `--ignore-gpu-blocklist` and `--enable-zero-copy`.

### Issue 3: Xfce Windows Drawn Behind Top Panel on Resume / Unlock
* **Symptom**: Resuming from suspend or unlocking the session caused maximized windows to draw behind the top panel, requiring manual minimize-maximize cycles to restore proper boundaries.
* **Root Cause**:
  1. **Monitor Alignment Geometry**: In `~/.config/xfce4/xfconf/xfce-perchannel-xml/displays.xml`, `DP-2` had an accidental offset of `Y = 73`, while `eDP-1` had `Y = 0`. This created a 73-pixel vertical dead zone at the top of the primary monitor.
  2. **Strut Race Condition**: On resume, `xfwm4` recalculated window geometries before `xfce4-panel` finished re-registering its partial strut (`_NET_WM_STRUT_PARTIAL`). With `borderless_maximize` enabled, windows expanded to the full monitor height `(Y=73 to 2233)` ignoring the panel area.

### Issue 4: Night Shift (Redshift) Intermittency, Gradual Transitions & Display Drops
* **Symptom**: Night Shift behaved unpredictably: it required pressing the shortcut multiple times, applied gradual/intermediate temperature transitions, changed intensity depending on the time of day, and disappeared upon waking or unlocking (often on only one of the two displays).
* **Root Cause**:
  1. **Background Transition Daemon & Autostart**: In `~/.config/autostart/redshift.desktop` and `redshift.conf`, an automated time-based daemon was running with `transition=1`, `temp-day=5800`, and `dawn-time=05:45-06:15`. The daemon was constantly calculating sun elevation and pulling the color temperature back toward daytime values, fighting the user's manual toggle.
  2. **Double-Toggle Bug**: The old script checked `pgrep -x redshift` or a stale state file rather than actual RandR CRTC hardware gamma. When waking or unlocking, the Nouveau driver or X11 reset CRTC gamma to neutral (1.0:1.0:1.0), but the script assumed Night Shift was still active, so the first shortcut press turned it "off", requiring a second press to turn it "on".
  3. **Multi-Monitor DisplayPort Link Training Lag**: `eDP-1` wakes in ~100ms, whereas `DP-2` (4K DisplayPort) requires ~1.2s for link training. Restoring gamma once immediately after wake applied the warm LUT to `eDP-1` while `DP-2` was still negotiating, leaving `DP-2` in neutral 6500K once it finally powered on.
* **Resolution**:
  - **Declarative User Configuration (`~/.config/night-shift.conf`)**: Configurable schedule (`START_TIME="16:00"`, `END_TIME="06:30"`), color temperature (`TEMP_NIGHT=2400`), and gamma (`GAMMA_NIGHT=0.85`).
  - **Instant RandR Hardware LUT**: Toggle ON executes `redshift -m randr -P -O "$TEMP_NIGHT" -g "$GAMMA_NIGHT"` (immediate 100% maximum intensity on all screens with zero fading). Toggle OFF runs `redshift -m randr -x` (immediate 6500K neutral).
  - **Direct Hardware Truth & Single-Touch Toggle**: `is_hardware_night_shift_active` queries `xrandr --verbose` directly (34ms). If the display is visually neutral, the first shortcut press activates it immediately.
  - **Automated Midnight-Spanning Schedule & Idempotency**: `toggle-redshift auto` handles windows spanning midnight (e.g. 16:00 to 06:30). If Night Shift is already active during scheduled hours, it does nothing (zero screen flicker, zero gamma re-application).
  - **Intelligent Manual Override Tracking**: Manual toggles during active or inactive windows are recorded in `~/.local/state/night-shift-override` and respected until the next period transition.
  - **Dual Sleep Timing & Watchdog**: `xfce-session-guard` restores color state at `0.6s` (internal panel) and `1.8s` (DP-2 4K), backed by a 30-second watchdog loop that auto-heals any desynced display.

### Issue 5: Unwanted 04:00 AM RTC Hardware Wakeups from Suspend
* **Symptom**: The laptop woke itself up from S3 RAM suspend at exactly `04:00:02 AM` every night.
* **Root Cause**:
  - `~/.local/bin/quota-ctl` had `"auto_4am_enabled": true` in `~/.local/state/quota_primer/state.json`.
  - The sleep hook `/lib/systemd/system-sleep/99-quota-primer` executed `quota-ctl on-suspend`, which called `/usr/sbin/rtcwake -m no -t <04:00 AM>`, arming the motherboard RTC BIOS alarm.

---

## 3. Implemented Architecture & Hacks

### A. Terminal & Compositor Tuning
1. **Ghostty (`~/.local/bin/ghostty` & `~/.config/ghostty/config`)**:
   - `GTK_A11Y=none`, `NO_AT_BRIDGE=1`, `GTK_MODULES=""`: Neutralizes the AT-SPI accessibility leak that saturated the GTK main thread event loop with 70+ minutes of CPU churn.
   - `GSK_RENDERER=gl`: Uses stable OpenGL renderer instead of buggy `ngl` in GTK4.
   - `scrollback-limit = 10000` & `image-storage-limit = 50000000`: Caps internal `PageList` pool to prevent memory inflation during agent streams.
   - `background-opacity = 1.0`: Native 24-bit TrueColor opaque window. Completely eliminates 4K alpha-compositing overhead.
   - `window-vsync = false`: Direct zero-latency keyboard input response.
2. **Picom (`~/.config/picom/picom.conf`)**:
   - `glx-no-rebind-pixmap = false`: Resolves texture cache incoherency and DRI3 memory leaks in Mesa Nouveau.
   - `xrender-sync-fence = true`: Uses XSyncFence to synchronize buffer swaps with GLX.
   - `rounded-corners-exclude`: Excludes `com.mitchellh.ghostty` and `Brave-browser` from corner-rounding shaders.

### B. Brave Browser Flag Pipeline (`~/.config/brave-flags.conf`)
Applied automatically via the wrapper `~/.local/bin/brave`:
```text
--disable-accelerated-2d-canvas
--disable-gpu-rasterization
--canvas-oop-rasterization=false
--enable-gpu-compositing
--ozone-platform=x11
--disable-features=Vulkan,UseChromeOSDirectVideoDecoder,AcceleratedVideoDecodeLinux
--enable-smooth-scrolling
```
* **Why this works**: 2D canvas animations (Matrix stream) and Skia rasterization run on the 8-thread Core i7 CPU (zero GPU channel deadlocks, zero orange shadows, 100% color accuracy, no clipboard freeze). Meanwhile, window compositing remains active on the GPU over standard X11 MIT-SHM, delivering **fluid 60 FPS video playback and complete desktop stability**.

### C. Display Geometry Alignment
- Aligned `DP-2` position in Xfconf to `X=0, Y=0`.
- Both monitors now align cleanly at `Y=0`.
- The panel occupies `Y=0` to `Y=33`; `_NET_WORKAREA` cleanly starts at `Y=33` across the entire 4K display.

### D. Session Guard & Instant Binary Night Shift (Automated & Configurable)
1. **User Configuration (`~/.config/night-shift.conf`)**:
   - `START_TIME="16:00"` & `END_TIME="06:30"`: Default schedule spanning midnight.
   - `TEMP_NIGHT=2400` & `GAMMA_NIGHT=0.85`: High-contrast blue-light blocking.
   - `AUTO_SCHEDULE=1`: Automatic daemon mode (`0` for manual-only).
2. **`~/.local/bin/toggle-redshift`**:
   - 100% deterministic binary toggle: ON sets `redshift -m randr -P -O "$TEMP_NIGHT" -g "$GAMMA_NIGHT"` instantly; OFF sets `redshift -m randr -x` instantly.
   - Zero background daemons, zero fading, zero dawn/dusk transitions, zero solar math.
   - Direct hardware detection via `xrandr --verbose` ensures 1-touch responsiveness.
   - `auto` subcommand evaluates midnight crossing and enforces strict idempotency ("if already active, do nothing").
   - Records manual overrides in `~/.local/state/night-shift-override` so manual adjustments are preserved.
3. **`~/.local/bin/xfce-session-guard` & `xfce-session-guard.service`**:
   - Background user daemon listening to D-Bus signals for screensaver unlock (`org.xfce.ScreenSaver.ActiveChanged(false)`) and system resume (`org.freedesktop.login1.Manager.PrepareForSleep(false)`).
   - Dual-stage wake recovery (`0.6s` and `1.8s`) handles DisplayPort 4K link training latency, ensuring `DP-2` and `eDP-1` never desync.
   - 30-second watchdog loop calls `toggle-redshift auto` to seamlessly apply scheduled changes and auto-heal hotplugged screens.
   - Executes `xfwm4 --replace` to re-assert window geometry below the top panel.

### E. RTC Wakeup Disabling
- Executed `quota-ctl auto-4am off` and cleared the hardware wakealarm (`rtcwake -m disable`).
- Disabled `quota-primer.timer` and `quota-primer.service` in systemd user session.

---

## 4. Automation Script (`apply_hacks.sh`)

To deploy or restore all configurations on a clean setup, run:

```bash
./hacks/dell-precision-m6800/apply_hacks.sh
```

### Script Capabilities:
- Idempotent and safe to run multiple times.
- Anchors to the repository root.
- Re-generates `~/.config/picom/picom.conf`, `~/.config/brave-flags.conf`, and `~/.config/ghostty/config`.
- Installs `toggle-redshift`, `xfce-session-guard`, and registers the `xfce-session-guard.service` user unit.
- Re-aligns dual-monitor coordinates and resets top panel struts.
- Disables RTC auto-wake timers.

---

## 5. Recommended Web Streaming Configuration

The Haswell Core i7-4910MQ and Kepler Quadro K3100M **do not possess hardware decoding blocks for VP9 or AV1 codecs**.

For maximum efficiency on YouTube and streaming platforms:
1. Open Brave and install the extension **[enhanced-h264ify](https://chromewebstore.google.com/detail/enhanced-h264ify/omkfmpieigfdclheacbehaaiannllgbj)**.
2. Open extension options and check **Block VP9** and **Block AV1**.
3. YouTube will serve streams encoded in **H.264 (AVC)**, allowing effortless 1080p60 / 1440p playback with negligible CPU load and zero frame drops.
