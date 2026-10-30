# anx — Declarative Manifest Schema (Production Specification)

This specification defines the production schema for declarative packages, configurations, profiles, and machine hacks. All definitions reside in YAML files under `manifests/` (embedded via `//go:embed` for zero-dependency binary portability) and are dynamically overridable and extensible via `<repo>/manifests/` and `~/.config/anx/manifests.d/*.yaml` without recompiling Go source code.

---

## 1. Package Catalog Schema (`manifests/packages/*.yaml`)

Each package is declared with cross-distro package mappings and an ordered fallback cascade of installation strategies (`native`, `github_release`, `script`, `source`, `language_pkg`).

```yaml
packages:
  - id: "ripgrep"
    name: "ripgrep"
    category: "cli-tools"
    description: "Fast line-oriented search tool"
    binary_name: "rg"
    binary_aliases: ["ripgrep"]
    version_args: ["--version"]
    ecosystem: "cargo"
    osv_package: "ripgrep"
    os: ["linux", "darwin", "windows"]
    depends_on: []
    tags: ["core", "search", "cli"]
    desired_version: ">= 14.1.0"
    strategies:
      # Tier 1: Native package manager translation matrix with optional conditions & custom repositories
      - type: "native"
        when:
          desktops: ["gnome", "xfce", "all"]
          display_servers: ["wayland", "x11"]
        repositories:
          apt: ["ppa:custom/stable"]
          dnf: ["copr:user/repo"]
          brew: ["homebrew/cask"]
        packages:
          apt: ["ripgrep"]
          dnf: ["ripgrep"]
          pacman: ["ripgrep"]
          zypper: ["ripgrep"]
          apk: ["ripgrep"]
          brew: ["ripgrep"]
          pkg: ["ripgrep"]        # Android Termux
          winget: ["BurntSushi.ripgrep.MSVC"]
          scoop: ["ripgrep"]

      # Tier 2: Pre-compiled GitHub release binary with archive extraction
      - type: "github_release"
        github_release:
          repo: "BurntSushi/ripgrep"
          version: "latest"
          asset_patterns:
            linux_amd64: "ripgrep-.*-x86_64-unknown-linux-musl\\.tar\\.gz"
            linux_arm64: "ripgrep-.*-aarch64-unknown-linux-gnu\\.tar\\.gz"
            darwin_arm64: "ripgrep-.*-aarch64-apple-darwin\\.tar\\.gz"
            darwin_amd64: "ripgrep-.*-x86_64-apple-darwin\\.tar\\.gz"
            windows_amd64: "ripgrep-.*-x86_64-pc-windows-msvc\\.zip"
          binary_name: "rg"
          strip_components: 1
          chmod_x: true

      # Tier 3: Rust Cargo language package fallback
      - type: "language_pkg"
        when:
          requires_binary: "cargo"
        language_pkg:
          ecosystem: "cargo"
          package: "ripgrep"

  - id: "ghostty"
    name: "Ghostty"
    category: "terminal"
    description: "Fast, feature-rich, GPU-accelerated terminal emulator"
    binary_name: "ghostty"
    version_args: ["--version"]
    os: ["linux", "darwin"]
    strategies:
      - type: "native"
        when:
          os: ["darwin"]
        packages:
          brew: ["ghostty"]

      # Declarative from-source build with Zig
      - type: "source"
        when:
          os: ["linux"]
          requires_binary: "zig"
        source:
          repo_url: "https://github.com/ghostty-org/ghostty"
          branch: "main"
          build_cmd: ["zig", "build", "-Doptimize=ReleaseFast"]
          install_cmd: ["cp", "zig-out/bin/ghostty", "${HOME}/.local/bin/ghostty"]

  - id: "herdr"
    name: "Herdr Multiplexer"
    category: "terminal"
    description: "Modern AI agent and terminal multiplexer"
    binary_name: "herdr"
    strategies:
      # Curl direct installer script with verification
      - type: "script"
        script:
          curl_url: "https://herdr.dev/install.sh"
          install_cmd: ["bash", "-c", "curl -fsSL https://herdr.dev/install.sh | bash"]
          uninstall_cmd: ["rm", "-f", "${HOME}/.local/bin/herdr"]
          check_cmd: ["herdr", "--version"]
```

---

## 2. Managed Configuration Schema (`manifests/configs.yaml`)

Configuration assets map canonical files in `files/` to destination paths across operating systems, with support for 3-way merge, SmartAppend shell markers, and directory synchronization.

```yaml
configurations:
  - id: "ghostty-config"
    name: "Ghostty Terminal Configuration"
    group: "terminal"
    description: "Ghostty configuration launching Herdr workspace multiplexer"
    source: "files/ghostty"
    target: "~/.config/ghostty"
    targets_by_os:
      darwin: "~/Library/Application Support/com.mitchellh.ghostty"
    os: ["linux", "darwin"]
    terminals: ["ghostty"]
    strategy: "directory_sync"
    backup: true

  - id: "zshrc-loader"
    name: "Zsh User Loader Block (.zshrc)"
    group: "terminal"
    description: "Non-destructive marker block importing arturonavax/env"
    source: "files/zsh/.zshrc"
    target: "~/.zshrc"
    os: ["linux", "darwin"]
    shells: ["zsh"]
    strategy: "marker_block"
    marker_start: "# >>> arturonavax/env >>>"
    marker_end: "# <<< arturonavax/env <<<"
    backup: true

  - id: "xfce-rcmd-shortcuts"
    name: "XFCE rcmd Shortcuts & Extension"
    group: "osconfig"
    description: "XFCE4 keyboard shortcuts configuration for rcmd switcher"
    source: "files/rcmd/rcmd.conf"
    target: "~/.config/rcmd/rcmd.conf"
    os: ["linux"]
    desktops: ["xfce"]
    strategy: "three_way"
    post_sync_cmds:
      - ["rcmd", "--sync-xfce"]
    backup: true
```

---

## 3. Profiles Schema (`manifests/profiles.yaml`)

Profiles group packages and configurations into distinct developer personas with inheritance.

```yaml
profiles:
  - name: "minimal"
    description: "Essential shell and core utilities"
    packages: ["zsh", "starship", "fzf", "bat", "eza", "ripgrep", "fd"]
    configurations: ["zshrc-loader", "zsh-base", "starship-config"]

  - name: "developer"
    description: "Standard developer workstation environment"
    extends: "minimal"
    packages:
      - "neovim"
      - "ghostty"
      - "herdr"
      - "atuin"
      - "delta"
      - "difftastic"
      - "usql"
      - "tealdeer"
      - "rtk"
      - "go"
      - "node"
      - "rust"
      - "python"
    configurations:
      - "ghostty-config"
      - "herdr-config"
      - "atuin-config"
      - "nvim-modular"
```

---

## 4. Hardware & Workstation Hacks Schema (`manifests/hacks.yaml`)

Declarative hardware optimizations detected via DMI chassis/product, GPU driver, and desktop environment:

```yaml
id: "dell-precision-m6800"
name: "Dell Precision M6800 Workstation Suite"
category: "workstation"
description: "Resolves Ghostty DRI3 buffer stalls, Nouveau GPU lockups, and disables 04:00 AM RTC wakeups"
safe_by_opt_in: true
match:
  os: ["linux"]
  dmi_vendor: "Dell"
  dmi_product: "Precision M6800"
  gpu_driver: "nouveau"
  desktop_env: ["xfce"]
backup_paths:
  - "~/.config/ghostty/config"
  - "~/.config/picom/picom.conf"
steps:
  - id: "apply-m6800-workstation-hacks"
    name: "Apply Dell Precision M6800 Nouveau & Xfce Optimizations"
    script: "apply_hacks.sh"
    touches:
      - "~/.config/ghostty/config"
      - "~/.local/bin/ghostty"
    privileged: false
```
