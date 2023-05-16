# 🦊 env

Terminal-based development environment.

_**[:scroll: Tutorials here :scroll:](./docs/README.md)**_

---

1. **[Installation :sparkles:](#installation-sparkles)**
2. **[Features :ballot_box_with_check:](#features-ballot_box_with_check)**
3. **[Requirements](#requirements)**
4. **[Custom configuration :gear:](#custom-configuration-gear)**
5. **[More keyboard speed :keyboard:](#more-keyboard-speed-keyboard)**
6. **[Precaution and Backup :warning:](#precaution-and-backup-warning)**
7. **[Reminders](#reminders)**
8. **[Remote Scripts](#remote-scripts)**

## Installation :sparkles:

This environment is prepared to be installed on **GNU/Linux Ubuntu** and **MacOS**,
read/adapt the installation scripts if you have another distribution/operating system.

```sh
bash <(curl -fsSL "https://env.arturonavax.dev/install.sh") help
```

List of installation parameters:

- `requirements` / `r`: Install the necessary tools, languages and dependencies.

  Install basic dependencies (depending on the operating system), and the
  following: `pyenv`, `python3`, `pipx`, `cargo`, `rustc`, `fnm`, `node`,
  `npm`, `pnpm` and `go`

- `fonts` / `f`: Install patched mono fonts.
- `terminal` / `t`: Install the terminal, shell, prompt, tmux and terminal tools.
- `editor` / `e`: Install the editor (`lvim`) and development tools.
- `osconfig` / `o`: Configure the operationg system with personal preferences.
- `all` / `a`: Install and integrate all of the above.

### Aggressive new installation

Uninstall the main programs and delete the configuration and cache folders

```sh
./uninstall.sh && ./install.sh all
```

## Requirements

GNU/Linux requires `apt` and `snap` to be installed.

MacOS requires Homebrew (`brew`) to be installed.

If `pnpm` is installed, it will be used instead of `npm`.

If `pipx` is installed, it will be used instead of `python3 -m pip` for some installations.

_[Resolve EACCES permissions when installing packages globally](https://docs.npmjs.com/resolving-eacces-permissions-errors-when-installing-packages-globally)
to avoid error when installing packages with npm._

### For `terminal`

- `git`

### For `editor`

- `git`
- `make`
- `cargo`
- `python3`

  - `pip`

- `node` >= _v12.0.0_

  - `npm`

### For `fonts`

- `fc-cache` _(package `fontconfig`)_

### To install the extra tools

`terminal` and `editor` Install extra tools and utilities only if these
requirements are installed:

- `go` >= _v1.15.0_

- `python3`

  - `pip`

- `node` >= _v12.0.0_

  - `npm`

## Precaution and Backup :warning:

Read [`install.sh`](./install.sh) file before running on your system

This installer replaces configuration files, so it makes an automatic
backup of all current files that it replaces in `~/.arturonavax-env-backups/`

The backup script is: [`backup_config.sh`](./utils/backup_config.sh)

## Features :ballot_box_with_check:

_Screenshots of the environment in [screenshots/](./docs/screenshots/README.md)_

- **True Color** :rainbow:

- Patched Mono Font: [Caskaydia Cove](https://github.com/ryanoasis/nerd-fonts/tree/master/patched-fonts/CascadiaCode)

### Terminal :computer: ([`alacritty`](https://alacritty.org/))

- [`tmux`](https://github.com/tmux/tmux)

  - [Colorscheme](https://github.com/arturonavax/tmux-theme)

- [`zsh`](https://github.com/zsh-users/zsh)

  - [Starship Prompt](https://starship.rs/)

  - Syntax highlighting :flashlight::rainbow:

  - Autosuggestions :thought_balloon:

  - [FzF](https://github.com/junegunn/fzf) :mag: (`<Ctrl>r`)

### Editor :pencil: ([`lvim`](https://www.lunarvim.org/))

- Fast startup time :zap:

- Lazy-load of plugins 🦥

- LSP / Code Completion 🧠:thought_balloon:

  - Installer (`:LspInstall <language>`)

  - Automatic language installation! (config server `:LspSettings <server>`)

  - Autocomplete signatures

  - Hover documentation (`K`)

  - Snippets

- File search :mag::page_facing_up: (`<Space>f`)

- Word search :mag::abc: (`<Space>j`)

- Linters :flashlight:

- Formatters 🛠️

- Code Actions

- Code Lens

_**And much more :eyes:**_

## Synchronize with updates

There is a utility script to synchronize with the new configuration files
without having to perform the complete installation.

```sh
./utils/sync_config.sh help
```

List of synchronization parameters:

- `terminal` / `t`: Synchronize the Terminal settings.
- `editor` / `e`: Synchronize Editor (lvim) settings.
- `osconfig` / `o`: Configure the operating system with personal preferences.
- `plugins` / `p`: Synchronize Editor (lvim) plugins.
- `vscode` / `v`: Synchronize Visual Studio Code settings.
- `devtools` / `d`: Synchronizes the configuration of development tools.
- `all` / `a`: Synchronizes the configuration of all of the above.

## Custom configuration :gear:

To add custom configurations do it in the `~/.lunarvim.lua` file

For example, to add more plugins would be:

```lua
vim.list_extend(lvim.plugins,
    {
        "folke/lsp-colors.nvim",
        event = "BufRead",
    }
)

-- change leader key
lvim.leader = ","

-- change theme
lvim.colorscheme = "onedarker"
```

## More keyboard speed :keyboard:

If with `Ctrl-Alt-F3` you go to another terminal without graphic interface and
use Vim you will notice that the movement speed is higher, this is because our
graphic server configures the "delay rate" and "repeat rate" of our keyboard
with "slower" values... these are the values of how much we must wait for a key
to repeat and how much it repeats

When I noticed this I felt very slow in Vim, you can change the delay and repeat
rate with the following command:

```sh
# on linux:
xset r rate 200 30
```

## Transparency

Terminal transparency can be turned on and off using the `transparent` command
and its alias `tt`

## Dark and Light theme

By executing the command `darktheme` (aka `da`) or `lighttheme` (aka `li`) you
can activate the different themes

## Reminders

- The first save (`:w`) in a new project, or a first installation of this
  environment will be slow, because it is caching.

## Remote Scripts

There are utility scripts that can be executed remotely as they do not depend
on other local files, these scripts are in the [src/remotes/](./src/remotes/) folder.

Scripts starting with underscore (`_`) are made to be executed with the `source`
command.

The following subdomains redirect to [remote scripts folder](./src/remotes/):

- `env.arturonavax.dev`
- `env.arturonavax.com`
- `environment.arturonavax.dev`
- `environment.arturonavax.com`

Examples:

```sh
bash <(curl -fsSL "https://env.arturonavax.dev/install_golang.sh") -i
```

```sh
source <(curl -fsSL "https://env.arturonavax.dev/_vars_colors.sh" | cat)
```
