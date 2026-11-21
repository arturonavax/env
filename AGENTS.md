# AI Agent Engineering Guide — env

`arturonavax/env` is a **pure declarative configuration repository** for the standalone `arturonavax/anx` engine.

## Ownership Boundaries

- **`env` owns**: `manifests/` (packages, configs, profiles, tools, hacks, archives), `files/` (dotfiles), and `scripts/` (manifest-referenced installer/osconfig scripts).
- **`env` does NOT own**: Go source code (`cmd/`, `internal/`, `go.mod`), ANX installers (`install.*`, `uninstall.*`), Cloudflare Worker (`deploy/cloudflare/`), or legacy Bash archives. Those belong to `arturonavax/anx`.

## Non-Negotiable Rules

1. **Zero private/sensitive/personal data**: Never commit credentials, tokens, private emails, personal names/paths, or runtime state (`~/.config/herdr/` session files, `.cloudflare/`, `.wrangler/`, `.atl/`).
2. **Manifest integrity**: Every `source` in `manifests/configs.yaml` and every script referenced in `manifests/packages/*.yaml` must exist in this repository. Validate with `anx repo verify`.
3. **Non-destructive aliases**: Never alias POSIX utilities (`grep`, `find`, `cat`, `less`) to modern replacements.
4. **Dotfile isolation**: `files/herdr/` strictly contains `config.toml`; host `~/.config/herdr/` must remain a real directory.
