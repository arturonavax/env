# Production Edge Router: Cloudflare Worker Guide

Comprehensive architecture and deployment guide for serving `env.arturonavax.dev` as an intelligent, zero-latency edge distribution router using Cloudflare Workers.

---

## 1. Architectural Philosophy

Modern CLI tools like Docker (`get.docker.com`), Rustup (`sh.rustup.rs`), Bun (`bun.sh`), and UV (`astral.sh/uv`) distribute their setup scripts through edge routers rather than static file links.

By placing a Cloudflare Worker on `env.arturonavax.dev/*`:

1. **Intelligent User-Agent Detection**:
   - `curl` or `wget` without path -> automatically delivers `install.sh` (`text/plain; charset=utf-8`).
   - PowerShell (`irm`) without path -> automatically delivers `install.ps1`.
   - Web Browsers -> redirects automatically to `https://arturonavax.dev/env` to share primary site styles and branding.
   - Discord / Slack / Telegram crawlers -> serves rich OpenGraph metadata preview cards.
2. **Edge Caching & Resiliency**:
   - Eliminates GitHub raw rate limits by caching upstream script responses in Cloudflare's global edge network for 1 hour with `stale-while-revalidate`.
3. **Strict Content-Type Negotiation**:
   - Explicitly sets `Content-Type: text/plain; charset=utf-8` and `X-Content-Type-Options: nosniff` to guarantee clean piped execution in POSIX shells (`sh`, `dash`, `ash`, `bash`, `zsh`) and Windows PowerShell.

---

## 2. Universal Edge Routing Logic

```mermaid
flowchart TD
    A["Incoming Request to env.arturonavax.dev"] --> B{"Has Path? (/install.sh, /install.ps1, etc.)"}
    B -- Yes --> C["Fetch Specified Raw File from GitHub with Edge Cache"]
    B -- No --> D{"Analyze User-Agent"}
    D -- "PowerShell / WindowsPowerShell / pwsh" --> E["Serve install.ps1 (text/plain)"]
    D -- "curl / wget / httpie / fetch / sh" --> F["Serve install.sh (text/plain)"]
    D -- "Browser (Mozilla, Chrome, Safari, etc.)" --> G["Redirect 302 to https://arturonavax.dev/env"]
    D -- "Social Bot (Discord, Slack, Twitter, Telegram)" --> H["Serve OpenGraph Preview Metadata"]
```

---

## 3. Production Worker Implementation (`worker.ts`)

Create this file in your Cloudflare Worker project (`src/worker.ts`):

```typescript
/**
 * Cloudflare Worker: Intelligent Edge Router for arturonavax/env
 * Subdomain: env.arturonavax.dev
 */

interface Env {
  GITHUB_USER?: string;
  GITHUB_REPO?: string;
  DEFAULT_BRANCH?: string;
  PERSONAL_WEBSITE_URL?: string;
}

const GITHUB_USER = "arturonavax";
const GITHUB_REPO = "env";
const DEFAULT_BRANCH = "main";
const UPSTREAM_RAW_BASE = `https://raw.githubusercontent.com/${GITHUB_USER}/${GITHUB_REPO}/${DEFAULT_BRANCH}`;
const PERSONAL_WEBSITE_URL = "https://arturonavax.dev/env";

// Cache TTL: 5 minutes edge cache, 1 hour s-maxage, stale-while-revalidate for 24h
const CACHE_CONTROL_HEADER = "public, max-age=300, s-maxage=3600, stale-while-revalidate=86400";

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);
    const userAgent = request.headers.get("user-agent") || "";
    const lowerUA = userAgent.toLowerCase();

    // 1. Direct explicit file requests (e.g. /install.sh, /install.ps1, /install.bat)
    if (url.pathname !== "/" && url.pathname !== "") {
      return fetchUpstreamRaw(request, ctx, `${UPSTREAM_RAW_BASE}${url.pathname}`);
    }

    // 2. PowerShell / Windows Terminal detection
    if (
      lowerUA.includes("powershell") ||
      lowerUA.includes("pwsh") ||
      lowerUA.includes("windowspowershell")
    ) {
      return fetchUpstreamRaw(request, ctx, `${UPSTREAM_RAW_BASE}/install.ps1`);
    }

    // 3. Command-line downloaders (curl, wget, httpie, fetch, etc.)
    if (
      lowerUA.includes("curl") ||
      lowerUA.includes("wget") ||
      lowerUA.includes("httpie") ||
      lowerUA.includes("fetch") ||
      lowerUA.includes("aria2")
    ) {
      return fetchUpstreamRaw(request, ctx, `${UPSTREAM_RAW_BASE}/install.sh`);
    }

    // 4. Social crawlers (OpenGraph preview cards)
    if (
      lowerUA.includes("discordbot") ||
      lowerUA.includes("slackbot") ||
      lowerUA.includes("twitterbot") ||
      lowerUA.includes("telegrambot")
    ) {
      return serveSocialMetadata();
    }

    // 5. Interactive Desktop / Mobile Web Browsers -> Redirect to personal website (/env)
    return Response.redirect(PERSONAL_WEBSITE_URL, 302);
  },
};

/**
 * Fetches script from GitHub Raw with Cloudflare Edge Caching and strict security headers
 */
async function fetchUpstreamRaw(
  request: Request,
  ctx: ExecutionContext,
  targetUrl: string
): Promise<Response> {
  const cache = caches.default;
  const cacheKey = new Request(targetUrl, { method: "GET" });

  let response = await cache.match(cacheKey);
  if (!response) {
    const upstreamRes = await fetch(targetUrl, {
      headers: {
        "User-Agent": "Cloudflare-Worker-ANX-Installer-Router",
      },
    });

    if (!upstreamRes.ok) {
      return new Response(`[Router Error]: Failed to fetch installer script from upstream (${upstreamRes.status})`, {
        status: 502,
        headers: { "Content-Type": "text/plain; charset=utf-8" },
      });
    }

    const body = await upstreamRes.text();
    response = new Response(body, {
      status: 200,
      headers: {
        "Content-Type": "text/plain; charset=utf-8",
        "Cache-Control": CACHE_CONTROL_HEADER,
        "X-Content-Type-Options": "nosniff",
        "Access-Control-Allow-Origin": "*",
      },
    });

    // Store in Cloudflare global edge cache
    ctx.waitUntil(cache.put(cacheKey, response.clone()));
  }

  return response;
}

/**
 * Serves OpenGraph metadata for messaging preview cards
 */
function serveSocialMetadata(): Response {
  const html = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>ANX Platform | Modular Dotfiles & Environment Orchestrator</title>
  <meta property="og:title" content="ANX Platform — arturonavax/env">
  <meta property="og:description" content="Production-ready multi-platform developer environment, modular dotfiles, hardware-aware power optimizations, and AI productivity tools.">
  <meta property="og:image" content="https://raw.githubusercontent.com/arturonavax/env/main/files/images/preview.png">
  <meta property="og:url" content="https://env.arturonavax.dev">
  <meta name="twitter:card" content="summary_large_image">
</head>
<body>
  <p>ANX Platform Installer Endpoint</p>
</body>
</html>`;
  return new Response(html, {
    headers: { "Content-Type": "text/html; charset=utf-8" },
  });
}
```

---

## 4. Wrangler Configuration (`wrangler.jsonc`)

Place this configuration file at the root of your Cloudflare Worker project:

```jsonc
{
  "$schema": "node_modules/wrangler/config-schema.json",
  "name": "env-installer-router",
  "main": "src/worker.ts",
  "compatibility_date": "2026-10-01",
  "routes": [
    {
      "pattern": "env.arturonavax.dev/*",
      "zone_name": "arturonavax.dev"
    }
  ],
  "observability": {
    "enabled": true
  }
}
```

---

## 5. Deployment Instructions

Authenticate with your Cloudflare account:

```bash
npx wrangler login
```

Test worker routing locally:

```bash
npx wrangler dev
```

Deploy the worker to the production edge:

```bash
npx wrangler deploy
```

Once deployed, all users accessing `env.arturonavax.dev` receive optimal, operating-system-tailored installer streams with zero friction and maximum speed, while browser visitors land directly on `https://arturonavax.dev/env`.
