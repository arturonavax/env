# anx — Cross-Platform Compatibility Matrix

`anx` is engineered to provide identical core capabilities across all major developer operating systems.

## 1. Platform Matrix

| Platform | Tier | Detection | Package Providers | Dotfile Destination | Caveats / Mitigations |
|---|---|---|---|---|---|
| **Linux (Debian/Ubuntu)** | Tier 1 | `/etc/os-release` (`ID=debian`/`ubuntu`) | `apt`, `binary`, `git` | `~/.config/`, `~/.local/` | Requires `sudo` for `apt`; static binaries preferred |
| **Linux (Arch)** | Tier 1 | `/etc/os-release` (`ID=arch`) | `pacman`, `binary`, `git` | `~/.config/`, `~/.local/` | Pacman handles bleeding-edge packages |
| **Linux (Fedora/RHEL)** | Tier 1 | `/etc/os-release` (`ID=fedora`) | `dnf`, `binary`, `git` | `~/.config/`, `~/.local/` | DNF5 / DNF4 command translation |
| **macOS (Apple Silicon / Intel)** | Tier 1 | `runtime.GOOS == "darwin"` | `brew`, `binary`, `git` | `~/.config/`, `~/Library/` | ARM64 vs AMD64 binary asset resolution |
| **Windows 10/11 Native** | Tier 1 | `runtime.GOOS == "windows"` | `binary`, `winget` (future) | `%USERPROFILE%\.config\`, `%APPDATA%` | No symlink assumptions without developer mode; atomic file lock handling on self-update |
| **Windows WSL2** | Tier 1 | `/proc/version` contains `microsoft` | `apt`, `pacman`, `binary` | Linux `$HOME` | Interop with Windows host browser |
| **Android Termux** | Tier 2 | `$PREFIX` contains `com.termux` | `pkg` (apt frontend), `binary` | `/data/data/com.termux/files/home` | No `sudo` or `/usr/bin`; narrow terminal screen handling |

## 2. Windows-Specific Adaptations

- Executable binaries use `.exe` extension automatically.
- Self-update utilizes staged rename (`anx.exe` -> `anx.old.exe`, new binary written to `anx.exe`, `anx.old.exe` scheduled for deletion upon exit).
- Line endings: configuration templates normalize LF line endings for cross-shell tools (e.g. Starship, Neovim, Git), and CRLF only for native batch files.
