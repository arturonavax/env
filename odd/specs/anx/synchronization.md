# anx — Safe Dotfile Synchronization & Three-Way Merging

## 1. Synchronization Model

Dotfiles can be modified simultaneously in the repository (e.g. pulled from git) and locally by the user. Simple overwrite risks wiping out user modifications.

`anx` implements a **True Three-Way Merge**:
- **Base ($O$)**: The file content at the time of the last successful sync, stored in `state.json`.
- **Local ($A$)**: The current content in the user's filesystem (e.g. `~/.zshrc` or `~/.config/nvim/init.lua`).
- **Incoming ($B$)**: The desired content from `arturonavax/env/files/...`.

## 2. Decision Matrix

| Base vs Local | Base vs Incoming | Local vs Incoming | Classification | Resolution Action |
|---|---|---|---|---|
| Unchanged | Unchanged | Identical | Clean | No action needed |
| Unchanged | Modified | Different | IncomingUpdate | Fast-forward apply incoming |
| Modified | Unchanged | Different | LocalModified | Retain local modification, update base revision |
| Modified | Modified | Identical | Convergent | Both sides made identical edit; update base |
| Modified | Modified | Different | ConcurrentEdit | Run 3-way line diff; if non-overlapping merge cleanly; if overlapping trigger Conflict |

## 3. Conflict Resolution

When conflicts cannot be cleanly merged automatically:
1. **Interactive Prompt / TUI Diff Editor**: Shows side-by-side local, incoming, and common base with options:
   - `[k] Keep Local`: Preserve local edits, reject incoming.
   - `[t] Take Incoming`: Overwrite with incoming, saving local backup snapshot first.
   - `[m] Manual Merge`: Launch external `$EDITOR` with merge conflict markers.
   - `[s] Skip`: Postpone decision without modifying target.
2. In non-interactive mode (`--non-interactive`), conflicts halt execution with exit code `3` (`ExitConflict`) unless `--force` or `--keep-local` / `--take-incoming` is explicitly passed.
