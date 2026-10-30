# remotes

Scripts that can be called remotely, since they do not depend on other local files.

The following subdomains redirect to this folder:

- `env.arturonavax.dev`
- `env.arturonavax.com`
- `environment.arturonavax.dev`
- `environment.arturonavax.com`

## Environment installer (Interactive Wizard or Non-Interactive)

```sh
# Launch interactive step-by-step installation wizard (Recommended)
bash <(curl -fsSL "https://env.arturonavax.dev/install.sh") --wizard

# Or display command-line options
bash <(curl -fsSL "https://env.arturonavax.dev/install.sh") help
```

List of installation parameters:

- `wizard` / `-w`: Launch interactive step-by-step installation wizard with safety checks and rollback.
- `requirements` / `r`: Install the necessary tools, languages and dependencies.
- `fonts` / `f`: Install patched mono fonts.
- `terminal` / `t`: Install the terminal, shell, prompt, Herdr and terminal tools.
- `editor` / `e`: Install the editor (`nvim` - LazyVim) and development tools.
- `ai`: Install AI CLI assistants (`agy`), RTK token killer, MCP servers, and harnesses.
- `osconfig` / `o`: Configure the operating system with personal preferences.
- `all` / `a`: Install and integrate all of the above.

## Golang installer

```sh
bash <(curl -fsSL "https://env.arturonavax.dev/install_golang.sh") -i
```

List of installation flags:

- `-i`: Install latest version available.
- `-c`: Compiling from sources.

## Fonts installer

```sh
curl -fsSL "https://env.arturonavax.dev/install_fonts.sh" | bash
```

## Visual Studio Code extensions installer

```sh
curl -fsSL "https://env.arturonavax.dev/install_vscode_extensions.sh" | bash
```

## OS config

### GNU/Linux

```sh
curl -fsSL "https://env.arturonavax.dev/linux_osconfig.sh" | bash
```

### MacOS

```sh
curl -fsSL "https://env.arturonavax.dev/macos_osconfig.sh" | bash
```

## MacOS CommandLineTools (CLT) installer

```sh
curl -fsSL "https://env.arturonavax.dev/macos_install_clt.sh" | bash
```

## aitools · AI Productivity, Quota Management & Automated Workflows Suite

Unified AI suite providing rolling quota window monitoring, smart wakeup priming (compatible with `agy`, `claude`, `codex`, `opencode`, `pi`, and `llm`), hardware RTC auto-wake (04:00 AM) with auto-resuspend, and AI-powered Conventional Git Commit generation (`aitools commit` / `ai-commit` / `ai-cm`).

### Installation

```sh
curl -fsSL "https://env.arturonavax.dev/install_aitools.sh" | bash
```

Or locally via installer:

```sh
./src/ai/install_aitools.sh
```

### CLI Quick Reference

```sh
aitools quota              # View live quota table and reset countdowns
aitools watch              # Continuous terminal TUI monitor
aitools wake               # Prime expired or idle rolling quota windows
aitools commit             # Generate Conventional Git Commit from staged diff
aitools daemon status      # Check auto-wake and systemd status
aitools daemon auto-wake on # Enable 04:00 AM RTC hardware auto-wake
```
