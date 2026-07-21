/// <reference types="@cloudflare/workers-types" />

// @ts-ignore - WASM module import
import gatewayWasm from './wasm/gateway.wasm';

interface Env {
  DB: D1Database;
  KV: KVNamespace;
  VERSION: string;
  ADMIN_GITHUB_CLIENT_ID: string;
  ADMIN_GITHUB_CLIENT_SECRET: string;
  ADMIN_GITHUB_REDIRECT_URI: string;
  ADMIN_GITHUB_USERNAMES: string;
  SESSION_SECRET: string;
}

interface WASMExports {
  memory: WebAssembly.Memory;
  _start: () => void;
  ready: () => number;
  malloc: (size: number) => number;
  free: (ptr: number) => void;
  handleRequest: (
    methodPtr: number, methodLen: number,
    pathPtr: number, pathLen: number,
    headersPtr: number, headersLen: number,
    bodyPtr: number, bodyLen: number,
    remoteAddrPtr: number, remoteAddrLen: number,
  ) => bigint;
}

const encoder = new TextEncoder();
const decoder = new TextDecoder();

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    // --- Mutable state shared via closures ---
    let memBuf: ArrayBuffer;
    let wasmExports: WASMExports;
    let ioLog = '';

    // --- Helper functions ---
    function readStr(ptr: number, maxLen: number = 65536): string {
      const buf = new Uint8Array(memBuf, ptr, maxLen);
      let end = 0;
      while (end < maxLen && buf[end] !== 0) end++;
      return decoder.decode(buf.subarray(0, end));
    }

    function allocStr(str: string): number {
      const bytes = encoder.encode(str + '\0');
      const ptr = wasmExports.malloc(bytes.length);
      new Uint8Array(memBuf, ptr, bytes.length).set(bytes);
      return ptr;
    }

    const envMap: Record<string, string> = {
      GITHUB_CLIENT_ID: env.ADMIN_GITHUB_CLIENT_ID || '',
      GITHUB_CLIENT_SECRET: env.ADMIN_GITHUB_CLIENT_SECRET || '',
      ADMIN_GITHUB_USERNAMES: env.ADMIN_GITHUB_USERNAMES || '',
      GITHUB_REDIRECT_URI: env.ADMIN_GITHUB_REDIRECT_URI || '',
      SESSION_SECRET: env.SESSION_SECRET || '',
      VERSION: env.VERSION || '0.1.0',
    };

    // --- WASI import shims ---
    const wasiImports = {
      clock_time_get(_id: number, _precision: bigint, timePtr: number): number {
        new DataView(memBuf).setBigUint64(timePtr, BigInt(Date.now()) * 1_000_000n, true);
        return 0;
      },
      args_sizes_get(argcPtr: number, argvBufSizePtr: number): number {
        const v = new DataView(memBuf);
        v.setUint32(argcPtr, 0, true);
        v.setUint32(argvBufSizePtr, 0, true);
        return 0;
      },
      args_get(_argv: number, _argvBuf: number): number { return 0; },
      fd_close(_fd: number): number { return 0; },
      fd_read(_fd: number, _iovs: number, _iovsLen: number, _nwritten: number): number { return 8; },
      fd_write(fd: number, iovs: number, iovsLen: number, nwritten: number): number {
        if (fd !== 1 && fd !== 2) return 8;
        let total = 0;
        const dv = new DataView(memBuf);
        for (let i = 0; i < iovsLen; i++) {
          const ptr = dv.getUint32(iovs + i * 8, true);
          const len = dv.getUint32(iovs + i * 8 + 4, true);
          ioLog += decoder.decode(new Uint8Array(memBuf, ptr, len));
          total += len;
        }
        if (ioLog.length > 0 && (ioLog.includes('\n') || ioLog.length > 1024)) {
          console.log(ioLog); ioLog = '';
        }
        dv.setUint32(nwritten, total, true);
        return 0;
      },
      random_get(buf: number, bufLen: number): number {
        const bytes = new Uint8Array(bufLen);
        crypto.getRandomValues(bytes);
        new Uint8Array(memBuf, buf, bufLen).set(bytes);
        return 0;
      },
    };

    // --- Env import stubs (closures over mutable memBuf/wasmExports) ---
    const envImports = {
      dbQuery(sqlPtr: number, sqlLen: number, paramsPtr: number, paramsLen: number): bigint {
        // D1 queries are async — can't be called from sync WASM import.
        // For health endpoint, this isn't called.
        return BigInt(allocStr('[]'));
      },
      dbExec(sqlPtr: number, sqlLen: number, paramsPtr: number, paramsLen: number): bigint {
        return BigInt(allocStr(JSON.stringify({ rows_affected: 0, last_insert_id: 0 })));
      },
      cacheGet(keyPtr: number, keyLen: number): bigint {
        return BigInt(0); // not found
      },
      cacheSet(keyPtr: number, keyLen: number, valuePtr: number, valueLen: number, ttlSeconds: number): void {},
      cacheDel(keyPtr: number, keyLen: number): void {},
      envGet(keyPtr: number, keyLen: number): bigint {
        const key = readStr(keyPtr);
        const val = envMap[key] || '';
        if (!val) return BigInt(0);
        return BigInt(allocStr(val));
      },
      fetch(methodPtr: number, methodLen: number, urlPtr: number, urlLen: number, headersPtr: number, headersLen: number, bodyPtr: number, bodyLen: number): bigint {
        // async fetch not available in sync WASM import
        return BigInt(allocStr(JSON.stringify({ status_code: 502, headers: {}, body: 'fetch not available in sync context' })));
      },
    };

    // --- Instantiate WASM ---
    let instance: WebAssembly.Instance;
    try {
      instance = (await WebAssembly.instantiate(gatewayWasm, {
        wasi_snapshot_preview1: wasiImports,
        env: envImports,
      })).instance;
    } catch (e: any) {
      return new Response(JSON.stringify({ status: 'error', message: `WASM instantiate failed: ${e.message || e}` }), {
        status: 500, headers: { 'content-type': 'application/json' },
      });
    }

    wasmExports = instance.exports as unknown as WASMExports;
    memBuf = wasmExports.memory.buffer;

    // --- Initialize Go runtime ---
    try {
      wasmExports._start();
    } catch (e: any) {
      return new Response(JSON.stringify({ status: 'error', message: `WASM _start failed: ${e.message || e}` }), {
        status: 500, headers: { 'content-type': 'application/json' },
      });
    }

    // Flush any log output
    if (ioLog) { console.log(ioLog); ioLog = ''; }

    if (wasmExports.ready() !== 1) {
      return new Response(JSON.stringify({ status: 'error', message: 'WASM not ready' }), {
        status: 503, headers: { 'content-type': 'application/json' },
      });
    }

    // --- Routing ---
    const url = new URL(request.url);

    // Admin SPA fallback
    if (url.pathname.startsWith('/admin') && !url.pathname.startsWith('/admin/api/')) {
      return new Response(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/><title>Bendy File Gateway</title></head><body><div id="root"></div><p>Admin dashboard static files not built. Run: cd web && npm run build</p></body></html>`, {
        status: 200, headers: { 'content-type': 'text/html; charset=utf-8' },
      });
    }

    // GitHub OAuth intercept
    let body = await request.text() || '';
    if (url.pathname === '/admin/api/v1/auth/github' && request.method === 'POST') {
      try {
        const reqBody = JSON.parse(body);
        if (reqBody.code) {
          const tokenResp = await fetch('https://github.com/login/oauth/access_token', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded', 'Accept': 'application/json' },
            body: `client_id=${encodeURIComponent(env.ADMIN_GITHUB_CLIENT_ID || '')}&client_secret=${encodeURIComponent(env.ADMIN_GITHUB_CLIENT_SECRET || '')}&code=${encodeURIComponent(reqBody.code)}`,
          });
          const tokenData = await tokenResp.json() as Record<string, unknown>;
          if (tokenData.access_token) {
            const userResp = await fetch('https://api.github.com/user', {
              headers: { 'Authorization': `Bearer ${tokenData.access_token}`, 'Accept': 'application/vnd.github.v3+json' },
            });
            const ghUser = await userResp.json() as Record<string, unknown>;
            body = JSON.stringify({
              github_user: { id: ghUser.id, login: ghUser.login, name: ghUser.name || '', avatar_url: ghUser.avatar_url || '' },
            });
          }
        }
      } catch { /* pass through */ }
    }

    // Collect headers
    const reqHeaders: Record<string, string> = {};
    request.headers.forEach((value, key) => { reqHeaders[key] = value; });
    const remoteAddr = request.headers.get('cf-connecting-ip') || '127.0.0.1';

    // Pass to WASM
    const m = allocStr(request.method);
    const p = allocStr(url.pathname);
    const h = allocStr(JSON.stringify(reqHeaders));
    const b = allocStr(body);
    const a = allocStr(remoteAddr);

    let resultJSON: string;
    try {
      const retPtr = Number(wasmExports.handleRequest(
        m, request.method.length, p, url.pathname.length,
        h, JSON.stringify(reqHeaders).length, b, body.length, a, remoteAddr.length,
      ));
      resultJSON = readStr(retPtr);
    } catch (e: any) {
      return new Response(JSON.stringify({ status: 'error', message: `handleRequest failed: ${e.message || e}` }), {
        status: 500, headers: { 'content-type': 'application/json' },
      });
    }

    let result: { status_code?: number; headers?: Record<string, string>; body?: string };
    try {
      result = JSON.parse(resultJSON);
    } catch {
      return new Response('Invalid WASM response', { status: 500 });
    }

    const responseHeaders = new Headers();
    if (result.headers) {
      for (const [key, value] of Object.entries(result.headers)) {
        responseHeaders.set(key, value as string);
      }
    }

    return new Response(result.body || null, {
      status: result.status_code || 200,
      headers: responseHeaders,
    });
  },
};
