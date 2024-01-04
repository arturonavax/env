# MacOS

Enable FileVault (or Firmware Password if it is an Intel processor) to lock
Recovery Mode and encrypt the disk.

---

1. **[Default keyboard shortcuts](#default-keyboard-shortcuts)**
2. **[Best applications for MacOS](#best-applications-for-macos)**
3. **[Devs](#devs)**

## Default keyboard shortcuts

- App launcher: `Command-Space`

- Lock: `Ctrl-Command-Q`

- Show apps (Mission Control): `Ctrl-UpArrow` or Swipe up using three fingers
(or four fingers)

- Switch apps: `Command-Tab`

- Close app (Quit): `Command-Q` or `Option-Commnad-Esc` (Force)

- Refresh page (F5): `Command-R`

- Switch input source: `Ctrl-Space` or `Fn`

- Full screen window: `Ctrl-Command-F`

- New window: `Command-N`

- Minimize window: `Command-H`

- Screenshots:
  - Area (Clipboard): `Shift-Ctrl-Command-4`
  - Complete (Clipboard): `Shift-Ctrl-Command-3`
  - Area (File): `Shift-Command-4`
  - Complete (File): `Shift-Command-3`
  - Options: `Shift-Command-5`

- Grave accent (\`) or Tilde (\~): Key to the right of the Shift (ABC input source).

- Emojis: `Fn-E`

- Show hidden files when selecting a file in Finder: `Shift-Command-.`

- Switch desktop: `Ctrl-LeftArrow` or `Ctrl-RightArrow` or Swipe left/right using
three fingers (or four fingers).

- Switch desktop quickly: `Ctrl-1` or `Ctrl-2`... Configuring it in System Settings
-> Keyboard -> Keyboard Shortcuts -> Mission Control -> Mission Control -> Switch
to Desktop X (While the Desktops are active).

### [System configs (Requiring reboot)](../src/remotes/macos_config.sh)

## Personal list of applications for MacOS

Almost all of these applications are available in [Setapp subscriptions](https://setapp.com)
($107.88 USD/annual). Cost of all these apps excluding Raycast, Spark, Parallels
and CrossOver: $404.65 USD.

- Git GUI: [SourceTree](https://www.sourcetreeapp.com)

  - _Pricing: Free._

- Hiding icons in menu bar: [Bartender](https://www.macbartender.com/) or [iBar](https://apps.apple.com/us/app/ibar-menubar-icon-control-tool/id6443843900?mt=12)

  - _Pricing Bartender: $16 USD._
  - _Pricing iBar: Free._

- Spotlight replacement, and Window Manager, and Keyboard shortcut manager: [Raycast](https://www.raycast.com/)
or [Alfred](https://www.alfredapp.com)

  - _Pricing Raycast:_
    - _Free._
    - _Pro: $96 USD/annual._

  - _Pricing Alfred:_
    - _Free._
    - _Powerpack: $34 USD._

- App Switching: [AltTab](https://alt-tab-macos.netlify.app/)

  - _Pricing: Free._

- Video player: [IINA](https://iina.io)

  - _Pricing: Free._

- Time awareness: [Pandan](https://sindresorhus.com/pandan)

  - _Pricing: Free._

- Window Manager: [Amethyst](https://ianyh.com/amethyst/) or [Rectangle](https://rectangleapp.com/)

  - _Pricing Amethyst: Free._

  - _Pricing Rectangle:_
    - _Free._
    - _Pro: $9.99 USD._

  - Notes:
    - In the Dock, the apps that are "Options" -> "Assign To" in "All Desktop"
      mode do not work well in Amethyst.
    - If I use Amethyst, I use Raycast or Alfred to set up window keyboard shortcuts
      such as Rectangle.

- Monitor Manager: [Monitor Control](https://monitorcontrol.app)

  - _Pricing: Free._

- Relax sounds: [Noizio](https://noiz.io)

  - _Pricing:_
    - _Pro: $19.99 USD._
    - _Lite: Free._

- App updater: [Latest](https://max.codes/latest/) or [MacUpdater](https://www.corecode.io/macupdater/)

  - _Pricing Latest: Free._

  - _Pricing MacUpdater:_
    - _Standard Edition: $11.61 USD._
    - _Scanning Only: Free._

- App cleaner: [AppCleaner](https://freemacsoft.net/appcleaner/)

  - _Pricing: Free._

- Text recognition (OCR): [TextSniper](https://textsniper.app) or [Shottr](https://shottr.cc)

  - _Pricing TextSniper: $7.99 USD._
  - _Pricing Shottr: Free._

- Screenshots: [CleanShot X](https://cleanshot.com/) or [Shottr](https://shottr.cc)

  - _Pricing CleanShot X: $29 USD_
  - _Pricing Shottr: Free._

- System monitor: [iStat Menus](https://bjango.com/mac/istatmenus/) or [Stats](https://github.com/exelban/stats)

  - _Pricing iStat Menus: $14.27 USD._
  - _Pricing Stats: Free._

- Video downloader: [Downie](https://software.charliemonroe.net/downie/) or [VDown](https://en.vdownapp.com)

  - _Pricing Downnie: $19.99 USD._
  - _Pricing VDown: Free._

- Calendar in the interface: [Dato](https://sindresorhus.com/dato) or [Itsycal](https://www.mowglii.com/itsycal/)

  - _Pricing Dato: $15 USD aprox._
  - _Pricing Itsycal: Free._

- Battery Care: [AlDente](https://apphousekitchen.com/)

  - _Pricing:_
    - _Pro: $26.89 USD._
    - _Normal: Free._

  - Notes:
    - Set the battery charge limit to 80%, do not let the battery discharge more
      than 20%.
    - Hardware Battery Percentage, activated.
    - Automatic Discharge, activated.
    - Disable Sleep until Charge Limit, activated.
    - Sailing Mode, activated with a difference of 10%.
    - Heat Protection, activated with a temperature limit of 30-35°C.
    - Stop charging when powered off/app closed, activated.
    - Use the Calibration Mode every 1 or 2 weeks.

- Copy with formatting: [Pure Paste](https://sindresorhus.com/pure-paste)

  - _Pricing: Free._

- Night shift: [f.lux](https://justgetflux.com)

  - _Pricing: Free._

- Cliboard manager _(Consider Raycast or Alfred)_: [Maccy](https://maccy.app)

  - _Pricing: Free._

  - Note: Add password and sensitive data management applications to the blacklist
    of the clipboard manager.

- Developer tools: [DevToysMac](https://github.com/ObuchiYuki/DevToysMac)

  - _Pricing: Free._

- Snippet manager _(Consider Raycast or Alfred)_: [Rocket Typist](https://www.witt-software.com/rockettypist/)

  - _Pricing: $9.95 USD._

- Replacement option switcher: [One Switch](https://fireball.studio/oneswitch/)

  - _Pricing: $4.99 USD._

- Cleaner and Antivirus: [CleanMyMac X](https://macpaw.com/cleanmymac)

  - _Pricing: $89.95 USD._

- Information on other Apple devices: [AirBuddy](airbuddy.app)

  - _Pricing: $15.46 USD._

- Command Palette in any applicattions: [Paletro](https://appmakes.io/paletro)
or `Shift-Command-?`

  - _Pricing Paletro: $6.99 USD._

- Calculator _(Consider Raycast or Alfred)_: [Numi](https://numi.app/)

  - _Pricing Numi: $29.74 USD._

- Keyboard Shortcut Manager _(Consider Raycast or Alfred)_:

  - [Karabiner](https://karabiner-elements.pqrs.org/)

    - _Pricing: Free._

  - [Keyboard Maestro](https://www.keyboardmaestro.com/main/)
    - _Pricing: $42.84 USD._

- Emails: [Spark](https://sparkmailapp.com/) or Apple Mail

  - _Pricing Spark: $59.99 USD/annual._
  - _Pricing Apple Mail: Free._

- Virtualization:

  - Systems: [Parallels](https://www.parallels.com/)

    - _Pricing:_
      - _Standard Edition: $129.99 USD._
      - _Pro Edition: $119.99 USD/annual._

  - Applications and programs (High performance): [CrossOver](https://www.codeweavers.com/crossover/)

    - _Pricing:_
      - _CrossOver +: $64 USD/annual._
      - _CrossOver Life: $494 USD._

    - Improve [CrossOver](https://www.codeweavers.com/crossover/)
      compatibility: [CXPatcher](https://github.com/italomandara/CXPatcher)
      - _Pricing: Free._

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
