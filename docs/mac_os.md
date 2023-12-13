# MacOS

## Default keyboard shortcuts

- App launcher: `Command-Space`

- Switch input source: `Ctrl-Space`

- Switch apps: `Command-Tab`

- Full screen window: `Ctrl-Command-F`

- New window: `Command-N`

- Minimize window: `Command-H`

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
brew tap homebrew/cask-fonts
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

## Other applications for MacOS

- Spotlight replacement, and Window Manager, and Keyboard shortcut manager: [Raycast](https://www.raycast.com/)
  - _Pricing:_
    - _Free._
    - _Pro: $96 USD/annual._

- App Switching: [AltTab](https://alt-tab-macos.netlify.app/)
  - _Pricing: Free._

- Calendar in the interface: [Dato](https://sindresorhus.com/dato)
  - _Pricing: Free._

- Hiding icons in menu bar: [Bartender](https://www.macbartender.com/)
  - _Pricing: $16 USD._

- Replacement option switcher: [One Switch](https://fireball.studio/oneswitch/)
  - _Pricing: $4.99 USD._

- Cleaner and Antivirus: [CleanMyMac X](https://macpaw.com/cleanmymac)
  - _Pricing: $89.95 USD._

- Emails: [Spark](https://sparkmailapp.com/)
  - _Pricing: $59.99 USD/annual._

- Video player: [VLC](https://www.videolan.org/)
  - _Pricing: Free._

- Battery Care: [AlDente Pro](https://apphousekitchen.com/)
  - _Pricing: $26.42 USD._

- Screenshots: [CleanShot X](https://cleanshot.com/)
  - _Pricing: $29 USD._

- Calculator: [Numi](https://numi.app/)
  - _Pricing: $29.74 USD._

- Monitor Manager: [Monitor Control](https://monitorcontrol.app)
  - _Pricing: Free._

- Window Manager _(Consider Raycast)_: [Rectangle](https://rectangleapp.com/)
  - _Pricing:_
    - _Free._
    - _Pro: $9.99 USD._

- Keyboard Shortcut Manager _(Consider Raycast)_:

  - [Karabiner](https://karabiner-elements.pqrs.org/)
    - _Pricing: Free._

  - [Keyboard Maestro](https://www.keyboardmaestro.com/main/)
    - _Pricing: $42.84 USD._

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
