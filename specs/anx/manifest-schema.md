# anx — Declarative Manifest Schema

## 1. Package Catalog Schema

Packages and tools are declared in a structured catalog (embedded in Go binary with YAML override support at `~/.config/anx/catalog.yaml`).

```yaml
version: "1.0"

packages:
  - id: "ripgrep"
    name: "ripgrep"
    category: "cli-tools"
    description: "Fast line-oriented search tool"
    binary_name: "rg"
    version_cmd: ["rg", "--version"]
    version_regex: 'ripgrep\s+([0-9]+\.[0-9]+\.[0-9]+)'
    providers:
      apt:
        package: "ripgrep"
      dnf:
        package: "ripgrep"
      pacman:
        package: "ripgrep"
      brew:
        package: "ripgrep"
      pkg: # termux
        package: "ripgrep"
      binary:
        github_repo: "BurntSushi/ripgrep"
        asset_pattern: "ripgrep-${version}-${arch}-${os}.tar.gz"
        strip_components: 1

  - id: "neovim"
    name: "Neovim"
    category: "editor"
    description: "Vim-fork focused on extensibility and usability"
    binary_name: "nvim"
    version_cmd: ["nvim", "--version"]
    version_regex: 'NVIM\s+v([0-9]+\.[0-9]+\.[0-9]+)'
    providers:
      pacman:
        package: "neovim"
      brew:
        package: "neovim"
      binary:
        github_repo: "neovim/neovim"
        asset_pattern: "nvim-linux-${arch}.appimage"
```

## 2. Managed Configuration Schema

Managed files map repository files to target destination paths with ownership policies.

```yaml
version: "1.0"

configurations:
  - id: "nvim-config"
    name: "Neovim Configuration"
    source: "files/nvim"
    target: "~/.config/nvim"
    strategy: "symlink" # or "copy"
    backup: true

  - id: "zshrc-block"
    name: "Zshrc Environment Block"
    source: "src/lib/templates/zshrc_block.sh"
    target: "~/.zshrc"
    strategy: "smart-append"
    marker_start: "# >>> arturonavax/env >>>"
    marker_end: "# <<< arturonavax/env <<<"
    backup: true
```

## 3. Profiles Schema

Profiles group packages and configurations into distinct developer personas.

```yaml
profiles:
  minimal:
    description: "Essential shell and core utilities"
    packages: ["zsh", "starship", "fzf", "bat", "eza"]
    configurations: ["zshrc-block", "starship-config"]

  workstation:
    description: "Full development workstation with GUI tools and AI agents"
    extends: "minimal"
    packages: ["neovim", "ghostty", "herdr", "atuin", "ripgrep", "fd", "delta", "difftastic", "usql", "tealdeer", "rtk"]
    configurations: ["nvim-config", "ghostty-config", "herdr-config", "atuin-config"]
```
