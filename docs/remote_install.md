# Remote Installation Architecture & Subdomain Configuration

Comprehensive guide for deploying and configuring the zero-dependency remote bootstrap endpoints for `arturonavax/env` and the `env.arturonavax.dev` subdomain.

---

## 1. Universal Remote Installation Commands

`arturonavax/env` provides native, zero-dependency remote bootstrap pipelines tailored to each operating system's canonical shell.

### Linux & macOS (Bash / Zsh)

```bash
# Interactive guided setup (reconnects /dev/tty automatically when piped)
curl -fsSL https://env.arturonavax.dev/install.sh | bash

# Unattended non-interactive automated provisioning
curl -fsSL https://env.arturonavax.dev/install.sh | bash -s -- -y --profile developer

# Safe dry-run preview (shows planned packages and touched files)
curl -fsSL https://env.arturonavax.dev/install.sh | bash -s -- --plan
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

If you want users to be able to run `curl -fsSL https://env.arturonavax.dev | bash` without even typing `/install.sh`:

Deploy this lightweight Cloudflare Worker on `env.arturonavax.dev/*`:

```javascript
export default {
  async fetch(request) {
    const url = new URL(request.url);
    const userAgent = request.headers.get("user-agent") || "";
    const repoBase = "https://raw.githubusercontent.com/arturonavax/env/main";

    // 1. Explicit file requests (/install.sh, /install.ps1, /install.bat, etc.)
    if (url.pathname !== "/" && url.pathname !== "") {
      return fetch(`${repoBase}${url.pathname}`);
    }

    // 2. Root domain request: detect OS / Client
    if (userAgent.includes("PowerShell") || userAgent.includes("WindowsPowerShell")) {
      return fetch(`${repoBase}/install.ps1`);
    }

    // Default to Bash for curl, wget, Linux, macOS
    return fetch(`${repoBase}/install.sh`);
  }
};
```

With this Worker active:
- **Linux/macOS**: `curl -fsSL https://env.arturonavax.dev | bash`
- **Windows**: `irm https://env.arturonavax.dev | iex`
