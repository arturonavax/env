# MacOS

- Enable FileVault (or Firmware Password if it is an Intel processor) to lock
Recovery Mode and encrypt the disk.

- With the brightness increase key (F2), you can get more brightness than the
maximum brightness with the bar in the options.

  - By using the `Shift-Option` key you can shift up and down by quarter on the `F`
    keys such as brightness and volume.

- With the "Screen Sharing" app you can control other Macs.

---

1. **[Default keyboard shortcuts](#default-keyboard-shortcuts)**
2. **[Devs](#devs)**
3. **[Personal list of applications for MacOS](#personal-list-of-applications-for-macos)**

## Default keyboard shortcuts

- App launcher: `Command-Space`

- Lock: `Control-Command-Q`

- Show apps (Mission Control): `Control-UpArrow` or Swipe up using three fingers
(or four fingers)

- Switch apps: `Command-Tab`

- Close app (Quit): `Command-Q` or `Option-Command-Esc` (Force)

- Refresh page (F5 in Windows): `Command-R`

- Switch input source: `Control-Space` or `Fn`

- Full screen window: `Control-Command-F`

- New window: `Command-N`

- Minimize window: `Command-M`

- Screenshots (I reconfigure it as follows):

  - Area (File): `Shift-Command-4`
  - Area (Clipboard): `Shift-Control-Command-4`

    - Move selected area: Move while holding the `Space` key down.
    - Enlarge corners of the selected area: Move while holding the `Option`
      key down.
    - Select window: `Space`

  - Complete (File): `Shift-Command-3`
  - Complete (Clipboard): `Shift-Control-Command-3`
  - Options: `Shift-Command-5`

- Grave accent (\`) or Tilde (\~): Key to the right of the Shift (ABC input source).

- Supr: `Fn-Delete`

- Emojis: `Fn-E` or `Control-Command-Space`

- Show hidden files when selecting a file in Finder: `Shift-Command-.`

- Spaces:

  - With "Desktop & Dock -> Mission Control -> Display have separate Spaces" activated,
    on each external monitor I create at least 2 spaces and do not use the first
    one, so that when disconnecting the monitor, the windows are not merged with
    the first space of the main monitor.

  - The most frequent windows are assigned to "All desktops", in the Dock ->
    Right-click on the app -> Options -> Assign To -> All Desktops.

  - Switch spaces: `Control-LeftArrow` or `Control-RightArrow` or Swipe left/right
    using three fingers (or four fingers).

  - Switch spaces quickly: `Control-1` or `Control-2` or etc etc... Configuring
    it in System Settings -> Keyboard -> Keyboard Shortcuts -> Mission Control ->
    Mission Control -> Switch to Desktop X (While the Desktops are active).

  - Close is spaces more quickly: While in the Spaces bar, press `Option` to see
    all the close space buttons more quickly.

- When the "Enter to send message" does not work: `Fn-Enter`

- Zoom somewhere on the screen: Scroll while pressing the `Control` key.

  - Accesibility -> Zoom -> Use scroll gesture with modifier keys to zoom.

- Open video in Picture in Picture or other options: Double right click
  (two-finger click).

  - Move the Picture in Picture window more precisely: Move while pressing
    the `Command` key.

### [System configs (Requiring reboot)](../src/remotes/macos_config.sh)

## Devs

### Install the CommandLineTools (CLT)

#### GUI

```bash
xcode-select --install
```

#### Headless

```bash
curl -fsSL "https://env.arturonavax.dev/macos_install_clt.sh" | bash
```

### Install Homebrew

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### Config PATH, Homebrew (Shell config)

This script determines the Homebrew folder on both the new Apple Silicon
architecture and older ones.

```bash
if [[ "$(uname -s)" == "Darwin" ]]; then
    brewbin="/usr/local/bin/brew"
    [[ ! -f "$brewbin" ]] && brewbin="/opt/homebrew/bin/brew" # For Apple Silicon
    [[ -f "$brewbin" ]] && eval "$("$brewbin" shellenv)"
fi
```

### GNU/Linux and Ubuntu utils

GNU commands that have names equal to default commands in MacOS are
installed with the prefix "g", for example: `ggrep`.

```bash
# GNU/Linux base tools
brew install ca-certificates gnupg bash zsh vim nano less grep screen ed watch zip unzip gzip gcc make autoconf \
    automake cmake python3 git mercurial curl wget m4 byacc swig bison flex ffmpeg pkg-config llvm \
    jq htop wdiff tcpdump iftop rsync openssl openvpn gdb nasm binutils coreutils diffutils findutils util-linux

# GNU tools for MacOS only
brew install gnu-indent gnu-sed gnu-tar gnu-which gnutls gnu-getopt gawk gpatch lsusb

# GNU tools with different names between APT and Homebrew
brew install netcat openssh
```

### Config PATH, GNU utils prefixs (Not recommended)

When installing GNU utilities, they are installed with the
prefix "g" in the commands, to avoid this prefix, the libexec/gnubin
folders in the Homebrew folder of each command must be added to the PATH.

This can cause conflicts when overriding the GNU commands over the default
ones in MacOS, so it is advisable to get used to adding the prefix "g" to GNU commands.

It may be useful to run this script when executing scripts which are only prepared
for GNU/Linux systems and depend on GNU versions of the commands.

```bash
if [[ "$(command -v brew)" != "" ]]; then
    for bindir in "$(brew --prefix)/opt/"*{"/libexec/gnubin","/bin"}; do
        export PATH="$bindir:$PATH"
    done

    for mandir in "$(brew --prefix)/opt/"*{"/libexec/gnuman","/share/man/man1"}; do
        export MANPATH="$mandir:$MANPATH"
    done
fi
```

### Show commands installed by a Homebrew package

```bash
brew list <package>
# Example: brew list coreutils
```

### Install latest zsh

```bash
# After installing zsh with brew and configuring it in the PATH
chsh -s "$(which zsh)"
```

### Install nerd fonts

```bash
brew tap --force homebrew/cask-fonts
brew install --cask font-caskaydia-cove-nerd-font
```

### Install files

#### .pkg

```bash
installer -pkg file.pkg -target /
```

#### .app

```bash
sudo cp -rf file.app /Applications/.
```

#### .dmg

```bash
hdiutil attach file.dmg

# after installation
sudo hdiutil detach file.dmg
```

### Config DNS

#### List Network Services

```bash
networksetup -listallnetworkservices
```

#### Show current DNS

```bash
networksetup -getdnsservers <NetworkService>
```

#### Set DNS

```bash
# Cloudflare
networksetup -setdnsservers <NetworkService> 1.1.1.1 1.0.0.1 2606:4700:4700::1111 2606:4700:4700::1001
```

#### Remove DNS

```bash
networksetup -setdnsservers <NetworkService> empty
```

#### Flush DNS

```bash
sudo dscacheutil -flushcache
sudo killall -HUP mDNSResponder
sudo killall mDNSResponderHelper
```

## Personal list of applications for MacOS

### Costs

Some of these applications are available in [Setapp subscriptions][setapp]
($107.88 USD/annual).

Total applications: 42 (excluding Parallels and CrossOver).

- Free applications: 18.
- Paid applications: 24.

  - One-time payment: 22 apps (choosing Alfred over Raycast). ($476.6 USD approx)
    - Some applications require a payment to upgrade to new versions.
    - Dato: Only one year of updates.
    - iStats Menu: Only 6 months of weather data.

  - Subscription: 2 apps. ($179.99 USD/annual approx)

- Total applications in Setapp: 23.

  - Without Setapp: $377.25 USD + $179.99 USD/annual. (approx)
    - Some applications require a payment to upgrade to new versions.
    - Dato: Only one year of updates.
    - iStats Menu: Only 6 months of weather data.

  - With Setapp: $107.88 USD/annual
    - Updates and upgrades included.

All applications available on Setapp and paying for those that are not:
_$107.88 USD/annual (23 Setapp apps) + $119.34 USD approx (6 One-time payment apps)_

### List

#### Essentials

- Hiding icons in menu bar: [Bartender][bartender] or [iBar][ibar]

  - _Pricing Bartender: $16 USD._ (🔶 Available on Setapp)

  - _Pricing iBar: Free._

- Spotlight replacement, and Window Manager, and Keyboard shortcut manager: [Raycast][raycast]
or [Alfred][alfred]

  - _Pricing Alfred:_

    - _Powerpack: $34 USD._

    - _Normal: Free._

  - _Pricing Raycast:_

    - _Normal: Free._

    - _Pro: $96 USD/annual._

- App switching: [AltTab][alttab] and [rcmd](https://lowtechguys.com/rcmd/)

  - _Pricing AltTab: Free._

  - _Pricing rcmd: $12.99 USD._

    - Note: The same behavior can be achieved with [Karabiner-Elements][karabiner]

- Window manager: [Amethyst][amethyst] and [Rectangle][rectangle]

  - _Pricing Amethyst: Free._

  - _Pricing Rectangle:_

    - _Normal: Free._

    - _Pro: $9.99 USD._

  - Notes:

    - I use Amethyst only for development and workflow, with the option
      "Automatically float applications except those listed".

    - I use Rectangle for everything that is not part of the Amethyst layout.

    - In the Dock, the apps that are "Options" -> "Assign To" in "All Desktop"
      mode do not work well in Amethyst.

- App cleaner: [AppCleaner][appcleaner]

  - _Pricing: Free._

  - Note: SmartDelete, activated.

- App updater: [Latest][latest] or [MacUpdater][macupdater]

  - _Pricing MacUpdater:_

    - _Standard Edition: $11.61 USD._

    - _Scanning Only: Free._

  - _Pricing Latest: Free._

- Organization and documentation: [Notion][notion]

  - _Pricing: Free._

- Time Tracker: [Timing][timing] or [Toggl Track][toggltrack]

  - _Pricing Timing: $120 USD/annual._ (🔶 Available on Setapp)

  - _Pricing Toggle Track: Free._

- Habit Tracker: [Habitify][habitify]

  - _Pricing:_

    - _Normal: Free._

    - _Pro: $30 USD._

- Text recognition (OCR): [TextSniper][textsniper] or [Shottr][shottr]

  - _Pricing TextSniper: $7.99 USD._ (🔶 Available on Setapp)

  - _Pricing Shottr: Free._

  - Note: Set shortcut to `Control-Option-Command-0` (Like [CleanShot X][cleanshotx])

- Screenshots: [CleanShot X][cleanshotx] or [Shottr][shottr]

  - _Pricing CleanShot X: $29 USD_ (Only one year of updates) (🔶 Available on Setapp)

  - _Pricing Shottr: Free._

  - Note: Set shortcut to (Like [CleanShot X][cleanshotx]):

    - Area screenshot: `Control-Option-Command-1`

      - Move selected area: Move while holding the `Space` key down.
      - Enlarge corners of the selected area: Move while holding the `Option`
        key down.
      - Select window: `Space`

    - Repeat area screenshot: `Control-Option-Command-2`
    - Any window screenshot: `Control-Option-Command-3`
    - Fullscreen screenshot: `Control-Option-Command-4`
    - Active window screenshot: `Control-Option-Command-5`
    - Scrolling screenshot: `Control-Option-Command-6`

- Calendar in the interface: [Dato][dato] or [Itsycal][itsycal]

  - _Pricing Dato: $15 USD approx._ (🔶 Available on Setapp)

  - _Pricing Itsycal: Free._

- Battery Care: [AlDente][aldente]

  - _Pricing:_

    - _Pro: $26.89 USD._ (🔶 Available on Setapp)

    - _Normal: Free._

  - Notes:

    - Set the battery [charge limit to 80%][aldente-chargelimit], do not let the
      battery discharge more than 20%.
    - [Hardware Battery Percentage][aldente-hardwarebattery], activated.
    - [Automatic Discharge][aldente-automaticdischarge], activated.
    - [Disable Sleep until Charge Limit][aldente-disablesleep], activated.
    - [Sailing Mode][aldente-sailingmode], activated with a difference of 10%.
    - [Heat Protection][aldente-heatprotection], activated with a temperature limit
      of 30-35°C.
    - [Stop charging when powered off/app closed][aldente-stopcharging-poweroff],
      activated.
    - [Stop charging when sleeping][aldente-stopcharging-sleeping], activated.
    - Use the [Calibration Mode][aldente-calibrationmode] every 1 or 2 weeks.

- Cliboard manager _(Consider [Raycast][raycast] or [Alfred][alfred])_: [Maccy][maccy]

  - _Pricing: Free._

  - Note: Add password and sensitive data management applications to the blacklist
    of the clipboard manager.

- Cheat Sheet shortcuts: [CheatSheet][cheatsheet]

  - _Pricing: Free._

- Calculator _(Consider [Raycast][raycast] or [Alfred][alfred])_: [Numi][numi]

  - _Pricing: $29.74 USD._ (🔶 Available on Setapp)

- Emails: [Spark][spark] or Apple Mail

  - _Pricing Spark: $59.99 USD/annual._ (🔶 Available on Setapp)

  - _Pricing Apple Mail: Free._

- Git GUI: [SourceTree][sourcetree]

  - _Pricing: Free._

#### Optional

- System monitor: [iStat Menus][istatmenu] or [Stats][stats]

  - _Pricing iStat Menus: $14.27 USD._ (Only 6 months of weather data)
    (🔶 Available on Setapp)

  - _Pricing Stats: Free._

- Webcam preview: [Hand Mirror][handmirror] or Apple Photo Booth

  - _Pricing Hand Mirror:_

    - _Pro: $9 USD approx._ (🔶 Available on Setapp)

    - _Normal: Free._

  - _Pricing Apple Photo Booth: Free._

- Time awareness: [Pandan][pandan]

  - _Pricing: Free._

- Dark mode in the browser: [Noir][noir] or [Dark Reader][darkreader]

  - _Pricing Noir:_

    - _For Safari: $4.99 USD approx._

  - _Pricing Dark Reader:_

    - _For Safari: $4.99 USD approx._

    - _For everything else: Free._

- Video player: [IINA][iina]

  - _Pricing: Free._

- Drag and drop bar: [Dropzone][dropzone]

  - _Pricing:_

    - _Pro: $35 USD._ (🔶 Available on Setapp)

    - _Normal: Free._

- Monitor manager: [BetterDisplay][betterdisplay] or [Monitor Control][monitorcontrol]

  - _Pricing BetterDisplay:_

    - _Normal: Free._

    - _Pro: $21.42 USD._

    - Notes for external monitors:

      - "High Resolution (HiDPI)", activated.
      - "Edit the system configuration of this display model", activated.
        - "Enable smooth scaling", activated.
          - "Add near-native HiDPI resolution with smooth scaling", activated.

  - _Pricing Monitor Control: Free._

- Relax sounds: [Noizio][noizio] or MacOS Background Sounds (System Settings
-> Accessibility -> Audio -> Background Sounds)

  - _Pricing Noizio:_

    - _Lite: Free._

    - _Pro: $19.99 USD._ (🔶 Available on Setapp)

  - _Pricing MacOS Background sounds: Free._

- Video downloader: [Downie][downie] or [VDown][vdown]

  - _Pricing Downnie: $19.99 USD._ (🔶 Available on Setapp)

  - _Pricing VDown: Free._

- Copy with formatting: [Pure Paste][purepaste]

  - _Pricing: Free._

- Developer tools: [DevToysMac][devtoysmac]

  - _Pricing: Free._

- Option switch manager: [One Switch][oneswitch] or [SwitchManager][switchmanager]

  - _Pricing One Switch: $4.99 USD._ (🔶 Available on Setapp)

  - _Pricing SwitchManager: Free._

- Snippet manager _(Consider [Raycast][raycast] or [Alfred][alfred])_: [Rocket Typist][rockettypist]

  - _Pricing:_

    - _Pro: $19.99 USD._ (🔶 Available on Setapp)

    - _Normal: Free._

- Hidden desktop icons _(Consider [One Switch][oneswitch] or
[SwitchManager][switchmanager])_: [HiddenMe][hiddenme]

  - _Pricing HiddenMe: $3 USD approx._

- Cleaner and Antivirus: [CleanMyMac X][cleanmymacx]

  - _Pricing: $89.95 USD._ (🔶 Available on Setapp)

- Information on other Apple devices: [AirBuddy][airbuddy]

  - _Pricing: $15.46 USD._ (🔶 Available on Setapp)

- Command Palette in any applicattions: [Paletro][paletro]
or `Shift-Command-?`

  - _Pricing Paletro: $6.99 USD._ (🔶 Available on Setapp)

- Check security settings: [Pareto Security][paretosecurity]

  - _Pricing: $17 USD._ (🔶 Available on Setapp)

- Logitech devices manager: [Logi Options+][logioptionsplus]

  - _Pricing: Free._

- Smooth scrolling: [Mos][mos]

  - _Pricing: Free._

- Keyboard shortcut manager _(Consider [Raycast][raycast] or [Alfred][alfred])_:

  - [Karabiner-Elements][karabiner]

    - _Pricing: Free._

  - [Keyboard Maestro][keyboardmaestro]

    - _Pricing: $42.84 USD._

- Virtualization:

  - Systems: [Parallels][parallels]

    - _Pricing:_

      - _Standard Edition: $129.99 USD._

      - _Pro Edition: $119.99 USD/annual._

  - Applications and programs (High performance): [CrossOver][crossover]

    - _Pricing:_

      - _CrossOver +: $64 USD/annual._

      - _CrossOver Life: $494 USD._

    - Improve [CrossOver][crossover] compatibility: [CXPatcher][cxpatcher]

      - _Pricing: Free._

[setapp]: https://setapp.com
[sourcetree]: https://www.sourcetreeapp.com
[bartender]: https://www.macbartender.com/
[ibar]: https://apps.apple.com/us/app/ibar-menubar-icon-control-tool/id6443843900?mt=12
[raycast]: https://www.raycast.com/
[alfred]: https://www.alfredapp.com
[alttab]: https://alt-tab-macos.netlify.app/
[iina]: https://iina.io
[pandan]: https://sindresorhus.com/pandan
[amethyst]: https://ianyh.com/amethyst/
[rectangle]: https://rectangleapp.com/
[monitorcontrol]: https://monitorcontrol.app
[noizio]: https://noiz.io
[latest]: https://max.codes/latest/
[macupdater]: https://www.corecode.io/macupdater/
[appcleaner]: https://freemacsoft.net/appcleaner/
[notion]: https://www.notion.so
[timing]: https://timingapp.com
[toggltrack]: https://toggl.com/track/time-tracking-mac/
[habitify]: https://www.habitify.me
[textsniper]: https://textsniper.app
[shottr]: https://shottr.cc
[cleanshotx]: https://cleanshot.com/
[istatmenu]: https://bjango.com/mac/istatmenus/
[stats]: https://github.com/exelban/stats
[downie]: https://software.charliemonroe.net/downie/
[vdown]: https://en.vdownapp.com
[dato]: https://sindresorhus.com/dato
[itsycal]: https://www.mowglii.com/itsycal/
[aldente]: https://apphousekitchen.com/
[purepaste]: https://sindresorhus.com/pure-paste
[maccy]: https://github.com/p0deje/Maccy
[devtoysmac]: https://github.com/ObuchiYuki/DevToysMac
[cheatsheet]: https://cheatsheet-mac.en.softonic.com/mac
[dropzone]: https://aptonic.com
[oneswitch]: https://fireball.studio/oneswitch/
[switchmanager]: https://www.doyourdata.com/mac-manager/switch-manager.html
[rockettypist]: https://www.witt-software.com/rockettypist/
[handmirror]: https://handmirror.app
[hiddenme]: https://apps.apple.com/us/app/hiddenme/id467040476
[cleanmymacx]: https://macpaw.com/cleanmymac
[airbuddy]: https://airbuddy.app
[paletro]: https://appmakes.io/paletro
[numi]: https://numi.app/
[paretosecurity]: https://paretosecurity.com
[spark]: https://sparkmailapp.com/
[logioptionsplus]: https://www.logitech.com/en-us/software/logi-options-plus.html
[mos]: https://mos.caldis.me
[karabiner]: https://karabiner-elements.pqrs.org/
[keyboardmaestro]: https://www.keyboardmaestro.com/main/
[parallels]: https://www.parallels.com/
[crossover]: https://www.codeweavers.com/crossover/
[cxpatcher]: https://github.com/italomandara/CXPatcher

[aldente-sailingmode]: https://apphousekitchen.com/feature-explanation-sailing-mode/
[aldente-automaticdischarge]: https://apphousekitchen.com/feature-explanation-automatic-discharge/
[aldente-hardwarebattery]: https://apphousekitchen.com/feature-explanation-hardware-battery-percentage/
[aldente-disablesleep]: https://apphousekitchen.com/feature-explanation-disable-sleep-until-charge-limit/
[aldente-heatprotection]: https://apphousekitchen.com/feature-explanation-heat-protection/
[aldente-calibrationmode]: https://apphousekitchen.com/feature-explanation-calibration-mode-2/
[aldente-chargelimit]: https://apphousekitchen.com/feature-explanation-charge-limiter/
[aldente-stopcharging-poweroff]: https://apphousekitchen.com/feature-explanation-stop-charging-when-powered-off-app-closed/
[aldente-stopcharging-sleeping]: https://apphousekitchen.com/feature-explanation-stop-charging-when-sleeping/
[betterdisplay]: https://github.com/waydabber/BetterDisplay
[noir]: https://apps.apple.com/es/app/noir-dark-mode-for-safari/id1592917505
[darkreader]: https://darkreader.org
