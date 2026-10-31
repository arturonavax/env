/**
 * Cloudflare Worker: Intelligent Edge Router for ANX Platform (arturonavax/env)
 * Domains: sh.arturonavax.dev & env.arturonavax.dev
 */

export interface Env {
  GITHUB_USER?: string;
  GITHUB_REPO?: string;
  DEFAULT_BRANCH?: string;
  PERSONAL_WEBSITE_URL?: string;
}

const DEFAULT_GITHUB_USER = "arturonavax";
const DEFAULT_GITHUB_REPO = "env";
const DEFAULT_BRANCH_NAME = "main";
const DEFAULT_PERSONAL_WEBSITE = "https://arturonavax.dev/env";

const CACHE_CONTROL_HEADER = "public, max-age=300, s-maxage=3600, stale-while-revalidate=86400";

export default {
  async fetch(request: Request, env: Env, ctx?: ExecutionContext): Promise<Response> {
    const url = new URL(request.url);
    const userAgent = request.headers.get("user-agent") || "";
    const lowerUA = userAgent.toLowerCase();

    const githubUser = env.GITHUB_USER || DEFAULT_GITHUB_USER;
    const githubRepo = env.GITHUB_REPO || DEFAULT_GITHUB_REPO;
    const branch = env.DEFAULT_BRANCH || DEFAULT_BRANCH_NAME;
    const websiteUrl = env.PERSONAL_WEBSITE_URL || DEFAULT_PERSONAL_WEBSITE;
    const upstreamRawBase = `https://raw.githubusercontent.com/${githubUser}/${githubRepo}/${branch}`;

    // 1. Direct explicit file requests (e.g. /install.sh, /install.ps1, /install.bat)
    if (url.pathname !== "/" && url.pathname !== "") {
      const targetUrl = `${upstreamRawBase}${url.pathname}`;
      return fetchUpstreamRaw(request, ctx, targetUrl);
    }

    // 2. PowerShell detection
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
      lowerUA === "posix" ||
      url.hostname.startsWith("sh.")
    ) {
      const targetUrl = `${upstreamRawBase}/install.sh`;
      return fetchUpstreamRaw(request, ctx, targetUrl);
    }

    // 4. Social crawlers
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

    // 5. Desktop / mobile browsers -> redirect to docs site
    return Response.redirect(websiteUrl, 302);
  },
};

async function fetchUpstreamRaw(
  _request: Request,
  ctx: ExecutionContext | undefined,
  targetUrl: string
): Promise<Response> {
  const cacheKey = new Request(targetUrl, { method: "GET" });
  let cache: Cache | undefined;
  try {
    // caches.default is present in real Cloudflare Workers runtime
    // @ts-ignore
    cache = typeof caches !== "undefined" ? caches.default : undefined;
  } catch {
    cache = undefined;
  }

  if (cache) {
    const cachedResponse = await cache.match(cacheKey);
    if (cachedResponse) {
      const headers = new Headers(cachedResponse.headers);
      headers.set("X-Router-Cache", "HIT");
      return new Response(cachedResponse.body, {
        status: cachedResponse.status,
        statusText: cachedResponse.statusText,
        headers,
      });
    }
  }

  const upstreamRes = await fetch(targetUrl, {
    headers: {
      "User-Agent": "Cloudflare-Worker-ANX-Installer-Router (arturonavax/env)",
      "Accept": "text/plain, text/x-shellscript, */*",
    },
  });

  if (!upstreamRes.ok) {
    return new Response(
      `[Router Error]: Failed to fetch script from upstream (${upstreamRes.status} ${upstreamRes.statusText})\nTarget: ${targetUrl}\n`,
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
  const response = new Response(scriptBody, {
    status: 200,
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
      "Cache-Control": CACHE_CONTROL_HEADER,
      "X-Content-Type-Options": "nosniff",
      "Access-Control-Allow-Origin": "*",
      "X-Router-Cache": "MISS",
    },
  });

  if (cache && ctx && typeof ctx.waitUntil === "function") {
    ctx.waitUntil(cache.put(cacheKey, response.clone()));
  }

  return response;
}

function serveSocialMetadata(websiteUrl: string): Response {
  const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>ANX Platform | Modular Dotfiles & Developer Environment</title>
  <meta name="description" content="Production-ready multi-platform developer environment, modular dotfiles, and AI productivity tools.">
  <meta property="og:title" content="ANX Platform — arturonavax/env">
  <meta property="og:description" content="High-performance modular dotfiles, unified Go orchestrator, and AI tooling.">
  <meta property="og:url" content="https://env.arturonavax.dev">
  <meta property="og:type" content="website">
  <meta name="twitter:card" content="summary_large_image">
</head>
<body>
  <p>ANX Platform Installer Endpoint. Visit <a href="${websiteUrl}">${websiteUrl}</a> for documentation.</p>
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
