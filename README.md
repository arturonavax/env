# ENV — Pure Declarative Environment Configuration Repository

`arturonavax/env` is a lightweight, pure declarative configuration repository consumed by the standalone [**ANX**](https://github.com/arturonavax/anx) orchestration engine.

It contains **zero engine source code** — only declarative YAML manifests, dotfiles, hardware optimization definitions, and package installation scripts.

## Structure

```text
env/
├── manifests/
│   ├── packages/       # Multi-OS / multi-distro package declarations
│   ├── hacks/          # Opt-in hardware-specific workstation optimizations
│   ├── configs.yaml    # Declarative dotfile synchronization rules
│   ├── profiles.yaml   # Composable workstation profiles
│   ├── tools.yaml      # External & developer tool declarations
│   └── archives.yaml   # Retired components for safe cleanup
├── files/              # Canonical dotfiles (Neovim, Zsh, Ghostty, Herdr, Starship, Atuin, rcmd, etc.)
├── scripts/            # Package & OS configuration scripts referenced by manifests
├── README.md
└── AGENTS.md
```

## Usage with ANX

Install ANX and apply this configuration repository (default):

```sh
curl -fsSL https://sh.arturonavax.dev/install.sh | sh
```

Or use a local checkout / fork with an existing ANX installation:

```sh
anx repo set /path/to/env
anx repo verify
anx plan --profile developer
anx apply --profile developer
anx sync
```

### Optional: hledger

`anx install hledger` installs `hledger`, `hledger-ui`, and `hledger-web` into the ANX user bin directory using [Eget](https://github.com/zyedidia/eget). ANX uses Homebrew on macOS, Chocolatey on Windows, and Homebrew or the SHA-256-checked Eget bootstrap on Linux.

The [upstream binaries](https://hledger.org/install.html#official-binaries) cover Linux x64, macOS x64 and ARM64, and Windows x64 (including ARM64 via x64 emulation). Check the installation with `hledger --version`, `hledger-ui --version`, and `hledger-web --version`.

## Public Repository Policy

This repository is public and must contain **zero private, sensitive, or personal data** (no credentials, API tokens, personal emails, private hostnames, or local machine paths).
