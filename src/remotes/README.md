# remotes

Scripts that can be called remotely, since they do not depend on other local files.

The following subdomains redirect to this folder:

- `env.arturonavax.dev`
- `env.arturonavax.com`
- `environment.arturonavax.dev`
- `environment.arturonavax.com`

## Environment installer

```sh
bash <(curl -fsSL "https://env.arturonavax.dev/install.sh") help
```

List of installation parameters:

- `requirements` / `r`: Install the necessary tools, languages and dependencies.
- `fonts` / `f`: Install patched mono fonts.
- `terminal` / `t`: Install the terminal, shell, prompt, tmux and terminal tools.
- `editor` / `e`: Install the editor (`lvim`) and development tools.
- `osconfig` / `o`: Configure the operationg system with personal preferences.
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

## Visual Studio Code extensions installer

```sh
curl -fsSL "https://env.arturonavax.dev/install_vscode_extensions.sh" | bash
```
