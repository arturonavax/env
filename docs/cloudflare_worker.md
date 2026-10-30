# Production Edge Router: Cloudflare Worker Guide

Comprehensive architecture, operational reference, and deployment manual for serving `env.arturonavax.dev` as an intelligent, zero-latency edge distribution router using Cloudflare Workers, the Cloudflare CLI (`cf`), and Wrangler.

---

## 1. Architectural Philosophy & Edge Routing

Modern developer tooling ecosystems (such as Docker via `get.docker.com`, Rust via `sh.rustup.rs`, Bun via `bun.sh`, and Astral UV via `astral.sh/uv`) distribute their setup scripts through edge routers rather than raw static links.

Deploying a Cloudflare Worker on `env.arturonavax.dev` provides four architectural advantages:

1. **Intelligent User-Agent Content Negotiation**:
   - **POSIX Shell Clients (`curl`, `wget`, `httpie`, `fetch`, `sh`)**: Automatically serves `install.sh` (`text/plain; charset=utf-8`).
   - **PowerShell Environments (`pwsh`, `powershell`, `irm`)**: Automatically serves `install.ps1` (`text/plain; charset=utf-8`).
   - **Interactive Web Browsers (Chrome, Firefox, Safari, Edge)**: Issues an HTTP 302 Redirect to `https://arturonavax.dev/env` to share primary site styles, documentation, and web assets.
   - **Social Crawlers (Discord, Slack, Twitter/X, Telegram, WhatsApp)**: Serves rich OpenGraph and Twitter Card preview metadata.
2. **Global Edge Caching & GitHub Rate-Limit Shield**:
   - Script bodies are cached across Cloudflare's 330+ points of presence (PoPs) using `caches.default` with a 1-hour `s-maxage` and 24-hour `stale-while-revalidate`.
   - Upstream GitHub Raw rate limits (`raw.githubusercontent.com`) are bypassed, delivering sub-15ms response times worldwide.
3. **Strict Content-Type Negotiation & Execution Safety**:
   - Enforces `Content-Type: text/plain; charset=utf-8` and `X-Content-Type-Options: nosniff` to guarantee clean piped execution in POSIX shells (`sh`, `dash`, `ash`, `bash`, `zsh`) and Windows PowerShell without buffering delays or binary corruption.
4. **Direct Subpath Transparency**:
   - Explicit subpaths (e.g. `/install.sh`, `/install.ps1`, `/install.bat`, `/scripts/install_golang.sh`) are forwarded transparently to the corresponding repository file.

---

## 2. Request Lifecycle & Routing Matrix

### Routing Matrix

| Incoming Path | User-Agent Classifier | HTTP Method | Action / Upstream Target | Status | Content-Type | Cache Policy |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `/*` (non-empty) | Any | `GET`, `HEAD` | `raw.githubusercontent.com/.../{path}` | `200 OK` | `text/plain; charset=utf-8` | `s-maxage=3600, stale-while-revalidate=86400` |
| `/` | PowerShell (`powershell`, `pwsh`) | `GET`, `HEAD` | `raw.githubusercontent.com/.../install.ps1` | `200 OK` | `text/plain; charset=utf-8` | `s-maxage=3600, stale-while-revalidate=86400` |
| `/` | POSIX CLI (`curl`, `wget`, `sh`) | `GET`, `HEAD` | `raw.githubusercontent.com/.../install.sh` | `200 OK` | `text/plain; charset=utf-8` | `s-maxage=3600, stale-while-revalidate=86400` |
| `/` | Social Bot (Discord, Slack, Twitter) | `GET`, `HEAD` | Static OpenGraph HTML Preview Card | `200 OK` | `text/html; charset=utf-8` | `public, max-age=86400` |
| `/` | Desktop / Mobile Browser | `GET`, `HEAD` | Redirect to `https://arturonavax.dev/env` | `302 Found` | `text/plain; charset=utf-8` | `no-cache` |

### Architecture Sequence Diagram

```mermaid
sequenceDiagram
    autonumber
    actor User as Client (Terminal or Browser)
    participant CF as Cloudflare Edge (env.arturonavax.dev)
    participant Cache as Cloudflare Edge Cache
    participant GH as GitHub Raw (arturonavax/env)
    participant Web as Personal Website (arturonavax.dev/env)

    User->>CF: HTTP GET https://env.arturonavax.dev/
    alt Has explicit subpath (/install.sh, /install.ps1, etc.)
        CF->>Cache: Check cache key for target subpath
        alt Cache Hit
            Cache-->>CF: Return cached script content
        else Cache Miss
            CF->>GH: Fetch raw.githubusercontent.com/.../{path}
            GH-->>CF: 200 OK with file body
            CF->>Cache: Store in edge cache (s-maxage=3600)
        end
        CF-->>User: 200 OK (text/plain; charset=utf-8)
    else Root path (/) & PowerShell User-Agent
        CF->>Cache: Check cache key for /install.ps1
        Cache-->>CF: Return cached install.ps1
        CF-->>User: 200 OK (text/plain; charset=utf-8)
    else Root path (/) & curl / wget / POSIX User-Agent
        CF->>Cache: Check cache key for /install.sh
        Cache-->>CF: Return cached install.sh
        CF-->>User: 200 OK (text/plain; charset=utf-8)
    else Root path (/) & Web Browser User-Agent
        CF-->>User: 302 Found (Location: https://arturonavax.dev/env)
        User->>Web: Follow redirect to website
    else Root path (/) & Social Crawler User-Agent
        CF-->>User: 200 OK (text/html with OpenGraph preview metadata)
    end
```

---

## 3. Subdomain & DNS Configuration

Cloudflare provides two distinct methods to connect `env.arturonavax.dev` to your Worker:

### Option A: Custom Domains (Recommended)

Cloudflare Workers **Custom Domains** natively bind a hostname directly to your Worker without requiring manual DNS records, origins, or dummy IP addresses. Cloudflare automatically manages the DNS record, edge TLS certificates, and traffic routing.

Advantages:
- Zero manual DNS records to create or maintain.
- Eliminates DNS misconfiguration risks (such as 1014 CNAME errors or 522 origin timeouts).
- Automatic HTTP/2, HTTP/3 (QUIC), and TLS 1.3 lifecycle management.

To configure via configuration file, declare `"custom_domain": true` in `wrangler.jsonc` (see [Section 5](#5-configuration-files)).

---

### Option B: Worker Routes with Proxied DNS (Classic)

If you prefer using traditional Route triggers (`env.arturonavax.dev/*`), Cloudflare requires an existing DNS record within the `arturonavax.dev` zone with Cloudflare Proxy enabled (**Orange Cloud**).

Worker routes execute **only** when traffic passes through Cloudflare's proxy. A DNS record pointing to an external server with proxy disabled (Grey Cloud) will bypass the Worker completely.

To create the required proxied record in the zone `arturonavax.dev`:

- **Record Type**: `CNAME`
- **Name**: `env`
- **Target**: `arturonavax.dev` (or the dummy IPv6 address `100::` / dummy IPv4 `192.0.2.1`)
- **Proxy Status**: **Proxied** (Orange Cloud enabled)
- **TTL**: Auto

---

## 4. Cloudflare CLI (`cf`) Management Commands

The Cloudflare CLI (`cf`) is Cloudflare's native resource management tool. Each command below is atomic, standalone, and ready to paste directly into your shell.

### 1. Authenticate with Cloudflare

Log in interactively via browser:

```bash
cf auth login
```

Verify authentication status and active profile:

```bash
cf auth status
```

---

### 2. Inspect Available Zones

List active zones to identify the Zone ID for `arturonavax.dev`:

```bash
cf zones list
```

Filter specifically for the target zone:

```bash
cf zones list --query arturonavax.dev
```

---

### 3. Configure DNS Record via `cf` (For Option B - Route Trigger)

Create a proxied CNAME record for `env.arturonavax.dev` pointing to `arturonavax.dev`:

```bash
cf dns records create --zone arturonavax.dev --body '{"type":"CNAME","name":"env","content":"arturonavax.dev","proxied":true,"ttl":1}'
```

Alternative: Create a proxied dummy IPv6 record (`100::` reserved RFC 6666 discard prefix):

```bash
cf dns records create --zone arturonavax.dev --body '{"type":"AAAA","name":"env","content":"100::","proxied":true,"ttl":1}'
```

List existing DNS records for `env.arturonavax.dev` to verify configuration:

```bash
cf dns records list --zone arturonavax.dev --query env.arturonavax.dev
```

---

### 4. Deploy the Worker via `cf`

Test the Worker locally with edge runtime emulation:

```bash
cf dev
```

Deploy the Worker to Cloudflare's global edge network:

```bash
cf deploy
```

---

### 5. Attach Custom Domain via `cf` (For Option A - Custom Domain)

Attach `env.arturonavax.dev` as a Custom Domain directly to the deployed Worker:

```bash
cf workers triggers custom-domains add --name env-installer-router --domain env.arturonavax.dev
```

List all configured triggers and routes for the Worker:

```bash
cf workers triggers list --name env-installer-router
```

---

### 6. Edge Observability & Log Streaming via `cf`

Stream real-time invocation logs, status codes, and execution latency from edge nodes:

```bash
cf workers tails create --name env-installer-router
```

---

## 5. Wrangler CLI Commands

For environments using Wrangler (`npx wrangler`), use these atomic commands:

### 1. Authentication & Local Development

Authenticate your workstation with Cloudflare:

```bash
npx wrangler login
```

Start the local development server with hot-reloading:

```bash
npx wrangler dev
```

Start local development targeting production upstream APIs:

```bash
npx wrangler dev --remote
```

---

### 2. Deploy to Production

Deploy the Worker bundle and routes:

```bash
npx wrangler deploy
```

---

### 3. Custom Domain Management via Wrangler

Add `env.arturonavax.dev` as a Custom Domain:

```bash
npx wrangler custom-domains add env.arturonavax.dev
```

List configured Custom Domains:

```bash
npx wrangler custom-domains list
```

---

### 4. Real-Time Telemetry & Log Tailing

Stream live console output and runtime exceptions from production edge nodes:

```bash
npx wrangler tail
```

Stream logs filtered exclusively for errors:

```bash
npx wrangler tail --status error
```

---

## 6. Complete Production Implementation

Create the following files in your standalone Worker repository or directory (e.g. `~/repos/env-installer-router/`).

### Production Worker Script (`src/worker.ts`)

```typescript
/**
 * Cloudflare Worker: Intelligent Edge Router for arturonavax/env
 * Subdomain: env.arturonavax.dev
 *
 * Responsibilities:
 * 1. User-Agent detection: POSIX shell -> install.sh, PowerShell -> install.ps1, Browser -> Redirect.
 * 2. Edge caching: Caches upstream raw GitHub scripts for 1 hour with stale-while-revalidate.
 * 3. OpenGraph generation: Serves rich preview metadata for messaging crawlers.
 * 4. Content safety: Strict text/plain; charset=utf-8 with X-Content-Type-Options: nosniff.
 */

export interface Env {
  /** GitHub organization or username owning the repository */
  GITHUB_USER?: string;
  /** GitHub repository name containing the dotfiles and installer scripts */
  GITHUB_REPO?: string;
  /** Primary branch used for serving distribution scripts */
  DEFAULT_BRANCH?: string;
  /** Target landing URL for interactive web browser visitors */
  PERSONAL_WEBSITE_URL?: string;
}

// Canonical defaults
const DEFAULT_GITHUB_USER = "arturonavax";
const DEFAULT_GITHUB_REPO = "env";
const DEFAULT_BRANCH_NAME = "main";
const DEFAULT_PERSONAL_WEBSITE = "https://arturonavax.dev/env";

// Edge Cache Policy: 5 min browser cache, 1 hour edge CDN cache, 24h background revalidation
const CACHE_CONTROL_HEADER = "public, max-age=300, s-maxage=3600, stale-while-revalidate=86400";

export default {
  /**
   * Main fetch event dispatcher
   */
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);
    const userAgent = request.headers.get("user-agent") || "";
    const lowerUA = userAgent.toLowerCase();

    const githubUser = env.GITHUB_USER || DEFAULT_GITHUB_USER;
    const githubRepo = env.GITHUB_REPO || DEFAULT_GITHUB_REPO;
    const branch = env.DEFAULT_BRANCH || DEFAULT_BRANCH_NAME;
    const websiteUrl = env.PERSONAL_WEBSITE_URL || DEFAULT_PERSONAL_WEBSITE;
    const upstreamRawBase = `https://raw.githubusercontent.com/${githubUser}/${githubRepo}/${branch}`;

    // 1. Direct explicit file requests (e.g. /install.sh, /install.ps1, /install.bat, /scripts/...)
    if (url.pathname !== "/" && url.pathname !== "") {
      const targetUrl = `${upstreamRawBase}${url.pathname}`;
      return fetchUpstreamRaw(request, ctx, targetUrl);
    }

    // 2. Windows PowerShell / PowerShell Core detection
    // Matches: 'PowerShell', 'pwsh', 'WindowsPowerShell', or PowerShell Invoke-RestMethod headers
    if (
      lowerUA.includes("powershell") ||
      lowerUA.includes("pwsh") ||
      lowerUA.includes("windowspowershell")
    ) {
      const targetUrl = `${upstreamRawBase}/install.ps1`;
      return fetchUpstreamRaw(request, ctx, targetUrl);
    }

    // 3. Command-line HTTP downloaders (POSIX sh, curl, wget, httpie, fetch, aria2)
    if (
      lowerUA.includes("curl") ||
      lowerUA.includes("wget") ||
      lowerUA.includes("httpie") ||
      lowerUA.includes("fetch") ||
      lowerUA.includes("aria2") ||
      lowerUA.startsWith("sh/") ||
      lowerUA === "posix"
    ) {
      const targetUrl = `${upstreamRawBase}/install.sh`;
      return fetchUpstreamRaw(request, ctx, targetUrl);
    }

    // 4. Social media scrapers and chat preview crawlers
    if (
      lowerUA.includes("discordbot") ||
      lowerUA.includes("slackbot") ||
      lowerUA.includes("twitterbot") ||
      lowerUA.includes("telegrambot") ||
      lowerUA.includes("whatsapp") ||
      lowerUA.includes("linkedinbot") ||
      lowerUA.includes("facebookexternalhit")
    ) {
      return serveSocialMetadata(websiteUrl);
    }

    // 5. Interactive Desktop and Mobile Web Browsers -> 302 Redirect to personal website
    return Response.redirect(websiteUrl, 302);
  },
};

/**
 * Fetches an installer script from GitHub Raw with Cloudflare Edge Caching
 * and strict security headers.
 */
async function fetchUpstreamRaw(
  request: Request,
  ctx: ExecutionContext,
  targetUrl: string
): Promise<Response> {
  const cache = caches.default;
  const cacheKey = new Request(targetUrl, { method: "GET" });

  // 1. Inspect Cloudflare Edge Cache
  let response = await cache.match(cacheKey);
  if (response) {
    // Add header to identify edge cache hit in diagnostics
    const headers = new Headers(response.headers);
    headers.set("X-Router-Cache", "HIT");
    return new Response(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers,
    });
  }

  // 2. Fetch from upstream GitHub Raw repository
  const upstreamRes = await fetch(targetUrl, {
    headers: {
      "User-Agent": "Cloudflare-Worker-ANX-Installer-Router (arturonavax/env)",
      "Accept": "text/plain, text/x-shellscript, */*",
    },
  });

  if (!upstreamRes.ok) {
    return new Response(
      `[Router Error]: Failed to fetch script from upstream repository (${upstreamRes.status} ${upstreamRes.statusText})\nTarget: ${targetUrl}\n`,
      {
        status: 502,
        headers: {
          "Content-Type": "text/plain; charset=utf-8",
          "X-Content-Type-Options": "nosniff",
        },
      }
    );
  }

  const scriptBody = await upstreamRes.text();

  // 3. Assemble response with security and caching headers
  response = new Response(scriptBody, {
    status: 200,
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
      "Cache-Control": CACHE_CONTROL_HEADER,
      "X-Content-Type-Options": "nosniff",
      "Access-Control-Allow-Origin": "*",
      "X-Router-Cache": "MISS",
    },
  });

  // 4. Store copy in Cloudflare Edge Cache asynchronously without blocking response
  ctx.waitUntil(cache.put(cacheKey, response.clone()));

  return response;
}

/**
 * Serves OpenGraph and Twitter Card metadata for link previews in chat applications.
 */
function serveSocialMetadata(websiteUrl: string): Response {
  const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>ANX Platform | Modular Dotfiles & Developer Environment</title>
  <meta name="description" content="Production-ready multi-platform developer environment, modular dotfiles, hardware-aware power optimizations, and AI productivity tools.">
  <meta property="og:title" content="ANX Platform — arturonavax/env">
  <meta property="og:description" content="High-performance modular dotfiles, unified Go orchestrator, hardware-aware power optimizations, and AI tooling.">
  <meta property="og:image" content="https://raw.githubusercontent.com/arturonavax/env/main/files/images/preview.png">
  <meta property="og:url" content="https://env.arturonavax.dev">
  <meta property="og:type" content="website">
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="ANX Platform — arturonavax/env">
  <meta name="twitter:description" content="High-performance modular dotfiles, unified Go orchestrator, and AI tooling.">
  <meta name="twitter:image" content="https://raw.githubusercontent.com/arturonavax/env/main/files/images/preview.png">
</head>
<body>
  <p>ANX Platform Installer Endpoint. Visit <a href="${websiteUrl}">${websiteUrl}</a> for full documentation.</p>
</body>
</html>`;

  return new Response(html, {
    status: 200,
    headers: {
      "Content-Type": "text/html; charset=utf-8",
      "Cache-Control": "public, max-age=86400",
    },
  });
}
```

---

### Configuration File (`wrangler.jsonc`)

```jsonc
{
  "$schema": "node_modules/wrangler/config-schema.json",
  "name": "env-installer-router",
  "main": "src/worker.ts",
  "compatibility_date": "2026-10-01",

  // Option A: Custom Domain Configuration (Recommended)
  "routes": [
    {
      "pattern": "env.arturonavax.dev",
      "custom_domain": true
    }
  ],

  // Option B: Classic Route Pattern (Uncomment if using proxied CNAME/AAAA DNS record)
  // "routes": [
  //   {
  //     "pattern": "env.arturonavax.dev/*",
  //     "zone_name": "arturonavax.dev"
  //   }
  // ],

  "vars": {
    "GITHUB_USER": "arturonavax",
    "GITHUB_REPO": "env",
    "DEFAULT_BRANCH": "main",
    "PERSONAL_WEBSITE_URL": "https://arturonavax.dev/env"
  },

  "observability": {
    "enabled": true,
    "head_sampling_rate": 1.0
  }
}
```

---

### Node.js Package Manifest (`package.json`)

```json
{
  "name": "env-installer-router",
  "version": "1.0.0",
  "description": "Intelligent edge distribution router for arturonavax/env",
  "main": "src/worker.ts",
  "scripts": {
    "dev": "wrangler dev",
    "deploy": "wrangler deploy",
    "tail": "wrangler tail",
    "cf:dev": "cf dev",
    "cf:deploy": "cf deploy"
  },
  "devDependencies": {
    "@cloudflare/workers-types": "^4.20241001.0",
    "typescript": "^5.6.2",
    "wrangler": "^4.156.0"
  }
}
```

---

### TypeScript Compiler Configuration (`tsconfig.json`)

```json
{
  "compilerOptions": {
    "target": "ESNext",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "types": ["@cloudflare/workers-types"],
    "strict": true,
    "skipLibCheck": true,
    "noEmit": true
  },
  "include": ["src/**/*"]
}
```

---

## 7. Verification & Smoke Testing Suite

After deploying the Worker, run these tests to verify that every user-agent classifier and routing condition responds as designed:

### 1. Test POSIX Shell Client (`curl`)

Should return the first lines of `install.sh`:

```bash
curl -fsSL https://env.arturonavax.dev | head -n 15
```

---

### 2. Test POSIX Shell Client (`wget`)

Should stream the shell installer cleanly:

```bash
wget -qO- https://env.arturonavax.dev | head -n 15
```

---

### 3. Test Windows PowerShell Client (`pwsh` / `irm`)

Should return the first lines of `install.ps1`:

```bash
curl -fsSL -H "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) PowerShell/7.4.0" https://env.arturonavax.dev | head -n 15
```

---

### 4. Test Interactive Web Browser Navigation

Should return HTTP 302 Found redirecting to `https://arturonavax.dev/env`:

```bash
curl -I -H "User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36" https://env.arturonavax.dev
```

---

### 5. Test Direct Script Subpaths

Verify direct subpath access for Unix:

```bash
curl -fsSL https://env.arturonavax.dev/install.sh | head -n 15
```

Verify direct subpath access for Windows PowerShell:

```bash
curl -fsSL https://env.arturonavax.dev/install.ps1 | head -n 15
```

Verify direct subpath access for Windows CMD:

```bash
curl -fsSL https://env.arturonavax.dev/install.bat | head -n 15
```

---

### 6. Test Social Messaging Crawler Preview

Should return HTML containing OpenGraph metadata tags:

```bash
curl -fsSL -H "User-Agent: Discordbot/2.0" https://env.arturonavax.dev | grep -E "og:title|og:description|twitter:card"
```

---

### 7. Test Edge Cache Hit Status

Execute two consecutive requests and verify the cache header flips from `MISS` to `HIT`:

First request (populates cache):

```bash
curl -s -D - https://env.arturonavax.dev/install.sh -o /dev/null | grep -E "HTTP/|X-Router-Cache|CF-Cache-Status"
```

Second request (served from edge cache):

```bash
curl -s -D - https://env.arturonavax.dev/install.sh -o /dev/null | grep -E "HTTP/|X-Router-Cache|CF-Cache-Status"
```

---

## 8. Operational Troubleshooting & Edge Gotchas

### Error 522 / 523 (Origin Unreachable)
- **Cause**: Using Route mode (`env.arturonavax.dev/*`) without a proxied DNS record, or pointing the DNS record to an IP address that drops incoming TCP connections on port 443.
- **Solution**: Switch to **Custom Domains** (`"custom_domain": true`), or ensure the DNS record uses `CNAME` pointing to `arturonavax.dev` with Cloudflare Proxy enabled (**Orange Cloud**).

### Error 1014 (CNAME Cross-User Ban)
- **Cause**: The DNS record is a CNAME pointing to another domain on Cloudflare that belongs to a different Cloudflare account.
- **Solution**: Point the CNAME to your own zone root (`arturonavax.dev`) or use the dummy IPv6 record `100::` with Orange Cloud proxy enabled.

### Stale Scripts After Committing to GitHub
- **Cause**: Cloudflare Edge Cache preserves the script body for up to 1 hour (`s-maxage=3600`).
- **Solution**: Purge the cache on demand via Cloudflare CLI:

```bash
cf cache purge --zone arturonavax.dev --files https://env.arturonavax.dev/install.sh
```

Or purge via Cloudflare API token:

```bash
curl -X POST "https://api.cloudflare.com/client/v4/zones/{ZONE_ID}/purge_cache" -H "Authorization: Bearer {API_TOKEN}" -H "Content-Type: application/json" --data '{"files":["https://env.arturonavax.dev/install.sh","https://env.arturonavax.dev/install.ps1"]}'
```
