# Remote Installation Architecture & Subdomain Configuration

Comprehensive guide for deploying and configuring the zero-dependency remote bootstrap endpoints for `arturonavax/env` and the `env.arturonavax.dev` subdomain.

---

## 1. Universal Remote Installation Commands

`arturonavax/env` provides native, zero-dependency remote bootstrap pipelines tailored to each operating system's canonical shell.

### Linux, macOS, BSD & Raspberry Pi (POSIX sh / bash / zsh / ash / dash)

```bash
# Interactive guided setup (reconnects /dev/tty automatically when piped; works with pure POSIX sh)
curl -fsSL https://env.arturonavax.dev/install.sh | sh

# Shorthand via Cloudflare Edge Worker
curl -fsSL https://env.arturonavax.dev | sh

# Unattended non-interactive automated provisioning
curl -fsSL https://env.arturonavax.dev/install.sh | sh -s -- -y --profile developer

# Safe dry-run preview (shows planned packages and touched files)
curl -fsSL https://env.arturonavax.dev/install.sh | sh -s -- --plan
```

### Windows (PowerShell)

```powershell
# Interactive Windows setup
irm https://env.arturonavax.dev/install.ps1 | iex

# Unattended automated setup
& ([scriptblock]::Create((irm https://env.arturonavax.dev/install.ps1))) -Yes -Profile developer
```

### Windows (Command Prompt - CMD)

```bat
curl -fsSL https://env.arturonavax.dev/install.bat -o %TEMP%\install.bat && %TEMP%\install.bat
```

---

## 2. Remote Bootstrap Engine Lifecycle

When executed remotely via `curl` or `irm`:

```mermaid
flowchart TD
    A["Remote Command (curl / irm)"] --> B{"Local Clone Present?"}
    B -- Yes --> C["Execute scripts/bootstrap_anx.sh (or .ps1)"]
    B -- No --> D{"git Installed?"}
    D -- Yes --> E["git clone --depth 1 https://github.com/arturonavax/env.git ~/.local/share/anx/repo"]
    D -- No --> F["Download repository archive (tar.gz / zip) via curl/wget/irm"]
    F --> G["Extract to ~/.local/share/anx/repo"]
    E --> C
    G --> C
    C --> H{"Pre-compiled release available?"}
    H -- Yes --> I["Download anx binary from GitHub Releases"]
    H -- No --> J["Bootstrap Go runtime & compile anx from source"]
    I --> K["Deploy to ~/.local/bin/anx & symlink aitools"]
    J --> K
    K --> L["Launch ANX Wizard / Apply Desired State"]
```

---

## 3. Subdomain Configuration (`env.arturonavax.dev`)

In previous iterations, the subdomain pointed to the legacy `src/remotes/` folder. With the unified ANX Platform, all bootstrap entry points reside cleanly at the repository root.

### Method A: Cloudflare URL Redirect Rule (Recommended - Zero Maintenance)

In your Cloudflare dashboard (or DNS provider):
1. Navigate to **Rules** -> **Redirect Rules** (or **Page Rules**).
2. Configure rule:
   - **Incoming Request**: `Hostname equals env.arturonavax.dev`
   - **Target URL**: `https://raw.githubusercontent.com/arturonavax/env/main/${1}`
   - **Status Code**: `302 Temporary Redirect` (or `301 Permanent`)

This enables:
- `https://env.arturonavax.dev/install.sh` -> points to `/install.sh`
- `https://env.arturonavax.dev/install.ps1` -> points to `/install.ps1`
- `https://env.arturonavax.dev/install.bat` -> points to `/install.bat`
- `https://env.arturonavax.dev/scripts/install_golang.sh` -> points to standalone Go installer

---

### Method B: Universal Cloudflare Worker (Ultimate UX with OS Detection)

If you want users to be able to run `curl -fsSL https://env.arturonavax.dev | sh` without even typing `/install.sh`:

Deploy the production Cloudflare Edge Worker on `env.arturonavax.dev/*` documented in detail in [docs/cloudflare_worker.md](file:///home/arthurnavah/repos/github.com/arturonavax/env/docs/cloudflare_worker.md).

With this Worker active:
- **Linux/macOS/BSD/Raspberry Pi/Android Termux**: `curl -fsSL https://env.arturonavax.dev | sh`
- **Windows (PowerShell)**: `irm https://env.arturonavax.dev | iex`
- **Desktop/Mobile Browsers**: Opens responsive HTML dashboard with one-click copy buttons.
