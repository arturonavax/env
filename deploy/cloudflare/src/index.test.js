import test from 'node:test';
import assert from 'node:assert';
import router from './index.ts';

test('PowerShell User-Agent routes to install.ps1', async () => {
  let fetchedUrl = '';
  globalThis.fetch = async (url) => {
    fetchedUrl = String(url);
    return new Response('# PowerShell script content', { status: 200 });
  };

  const req = new Request('https://env.arturonavax.dev/', {
    headers: { 'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) PowerShell/7.4.5' },
  });

  const res = await router.fetch(req, {});
  assert.strictEqual(res.status, 200);
  assert.ok(fetchedUrl.endsWith('/install.ps1'), `expected install.ps1, got ${fetchedUrl}`);
});

test('curl User-Agent routes to install.sh', async () => {
  let fetchedUrl = '';
  globalThis.fetch = async (url) => {
    fetchedUrl = String(url);
    return new Response('#!/bin/sh\necho "install"', { status: 200 });
  };

  const req = new Request('https://env.arturonavax.dev/', {
    headers: { 'User-Agent': 'curl/8.5.0' },
  });

  const res = await router.fetch(req, {});
  assert.strictEqual(res.status, 200);
  assert.ok(fetchedUrl.endsWith('/install.sh'), `expected install.sh, got ${fetchedUrl}`);
});

test('Browser User-Agent on env.arturonavax.dev redirects to docs site', async () => {
  const req = new Request('https://env.arturonavax.dev/', {
    headers: { 'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/130.0.0.0 Safari/537.36' },
  });

  const res = await router.fetch(req, {});
  assert.strictEqual(res.status, 302);
  assert.strictEqual(res.headers.get('Location'), 'https://arturonavax.dev/env');
});

test('Explicit subpath /install.bat is forwarded', async () => {
  let fetchedUrl = '';
  globalThis.fetch = async (url) => {
    fetchedUrl = String(url);
    return new Response('@echo off', { status: 200 });
  };

  const req = new Request('https://env.arturonavax.dev/install.bat', {
    headers: { 'User-Agent': 'curl/8.5.0' },
  });

  const res = await router.fetch(req, {});
  assert.strictEqual(res.status, 200);
  assert.ok(fetchedUrl.endsWith('/install.bat'), `expected install.bat, got ${fetchedUrl}`);
});

test('Bypass cache with ?nocache query parameter', async () => {
  let fetchCount = 0;
  globalThis.fetch = async () => {
    fetchCount++;
    return new Response('#!/bin/sh\n', { status: 200 });
  };

  const req = new Request('https://sh.arturonavax.dev/?nocache', {
    headers: { 'User-Agent': 'curl/8.5.0' },
  });

  const res = await router.fetch(req, {});
  assert.strictEqual(res.status, 200);
  assert.strictEqual(fetchCount, 1);
});
