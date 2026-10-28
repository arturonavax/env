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

## AI Quota Primer & RTC Auto-Wake Daemon (Linux / Xfce)

Intelligently primes rolling 5-hour quota windows for Antigravity (Gemini, Claude, GPT) via `agy -p /usage --output-format json` telemetry and manages hardware RTC alarms (04:00 AM) with auto-resuspend.

### Installation

```sh
curl -fsSL "https://env.arturonavax.dev/install_quota_daemon_xfce.sh" | bash
```

Or locally:

```sh
./src/remotes/install_quota_daemon_xfce.sh --install
```

### Uninstallation

```sh
curl -fsSL "https://env.arturonavax.dev/uninstall_quota_daemon_xfce.sh" | bash
```

Or with full configuration purge:

```sh
./src/remotes/uninstall_quota_daemon_xfce.sh --purge
```
