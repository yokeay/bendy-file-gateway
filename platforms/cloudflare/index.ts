/// <reference types="@cloudflare/workers-types" />

// @ts-ignore - WASM module import for wrangler
import gatewayWasm from '../wasm/gateway.wasm';

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
  _start?: () => void;
  _initialize?: () => void;
  ready: () => number;
  malloc: (size: number) => number;
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
    // Helpers that will be wired to WASM memory after instantiation
    let readCString: (ptr: number, maxLen?: number) => string = () => '';
    let allocString: (str: string) => number = () => 0;

    const envMap: Record<string, string> = {
      GITHUB_CLIENT_ID: env.ADMIN_GITHUB_CLIENT_ID || '',
      GITHUB_CLIENT_SECRET: env.ADMIN_GITHUB_CLIENT_SECRET || '',
      ADMIN_GITHUB_USERNAMES: env.ADMIN_GITHUB_USERNAMES || '',
      GITHUB_REDIRECT_URI: env.ADMIN_GITHUB_REDIRECT_URI || '',
      SESSION_SECRET: env.SESSION_SECRET || '',
      VERSION: env.VERSION || '0.1.0',
    };

    // The WASM binary defines and exports its own memory (does NOT import env.memory)
    const mod = await WebAssembly.instantiate(gatewayWasm, {
      env: {
        dbQuery: (_sp: number, _sl: number, _pp: number, _pl: number): bigint => BigInt(0),
        dbExec: (_sp: number, _sl: number, _pp: number, _pl: number): bigint => {
          return BigInt(allocString('{"rows_affected":0,"last_insert_id":0}'));
        },
        cacheGet: (_kp: number, _kl: number): bigint => BigInt(0),
        cacheSet: (_kp: number, _kl: number, _vp: number, _vl: number, _t: number): void => {},
        cacheDel: (_kp: number, _kl: number): void => {},
        envGet: (keyPtr: number, _keyLen: number): bigint => {
          const key = readCString(keyPtr);
          const val = envMap[key] || '';
          if (!val) return BigInt(0);
          return BigInt(allocString(val));
        },
        fetch: (_mp: number, _ml: number, _up: number, _ul: number, _hp: number, _hl: number, _bp: number, _bl: number): bigint => {
          return BigInt(allocString('{"status_code":500,"headers":{},"body":"fetch not available in WASM import"}'));
        },
      },
    });

    const exports = mod.instance.exports as unknown as WASMExports;
    const mem = exports.memory;

    // Wire helpers to use WASM's actual memory
    readCString = (ptr: number, maxLen: number = 65536): string => {
      if (ptr === 0) return '';
      const view = new Uint8Array(mem.buffer, ptr, maxLen);
      let len = 0;
      while (len < maxLen && view[len] !== 0) len++;
      return decoder.decode(view.subarray(0, len));
    };

    allocString = (str: string): number => {
      const bytes = encoder.encode(str + '\0');
      const ptr = exports.malloc(bytes.length);
      new Uint8Array(mem.buffer, ptr, bytes.length).set(bytes);
      return ptr;
    };

    // Initialize Go runtime (runs main() which sets up handlers and calls select{})
    const initFn = exports._initialize || exports._start;
    if (initFn) initFn();

    if (exports.ready() !== 1) {
      return new Response(JSON.stringify({ status: 'error', message: 'WASM not ready' }), {
        status: 503,
        headers: { 'content-type': 'application/json' },
      });
    }

    const url = new URL(request.url);
    const headers: Record<string, string> = {};
    request.headers.forEach((value, key) => { headers[key] = value; });
    let body = request.body ? await request.text() : '';
    const remoteAddr = request.headers.get('cf-connecting-ip') || '127.0.0.1';

    // Intercept GitHub OAuth: exchange code for user info before passing to WASM.
    // WASM can't make async fetch calls, so we do the OAuth dance here.
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
              github_user: {
                id: ghUser.id,
                login: ghUser.login,
                name: ghUser.name || '',
                avatar_url: ghUser.avatar_url || '',
              },
            });
          }
        }
      } catch {
        // Pass through to WASM — it will return an appropriate error
      }
    }

    const writeStr = (str: string): { ptr: number; len: number } => {
      const bytes = encoder.encode(str);
      const ptr = exports.malloc(bytes.length);
      new Uint8Array(mem.buffer, ptr, bytes.length).set(bytes);
      return { ptr, len: bytes.length };
    };

    const m = writeStr(request.method);
    const p = writeStr(url.pathname);
    const h = writeStr(JSON.stringify(headers));
    const b = writeStr(body);
    const a = writeStr(remoteAddr);

    const resultPtr = exports.handleRequest(m.ptr, m.len, p.ptr, p.len, h.ptr, h.len, b.ptr, b.len, a.ptr, a.len);

    const resultJSON = readCString(Number(resultPtr));
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
