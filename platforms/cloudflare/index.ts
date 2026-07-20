/// <reference types="@cloudflare/workers-types" />

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

// Memory buffer for WASM linear memory I/O
let memoryBuffer: ArrayBuffer;

const encoder = new TextEncoder();
const decoder = new TextDecoder();

// Allocate a string in WASM linear memory, returns pointer
function allocateString(memory: WebAssembly.Memory, str: string): number {
  const buf = encoder.encode(str + '\0');
  const ptr = memory.buffer.byteLength;
  // Grow memory if needed
  const newSize = ptr + buf.length;
  if (newSize > memory.buffer.byteLength) {
    memory.grow(Math.ceil((newSize - memory.buffer.byteLength) / 65536) + 1);
  }
  const view = new Uint8Array(memory.buffer, ptr, buf.length);
  view.set(buf);
  return ptr;
}

// Read a null-terminated string from WASM linear memory
function readString(memory: WebAssembly.Memory, ptr: number): string {
  const view = new Uint8Array(memory.buffer, ptr, 65536);
  let len = 0;
  while (view[len] !== 0 && len < 65536) len++;
  return decoder.decode(view.subarray(0, len));
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const mod = await WebAssembly.instantiate(gatewayWasm, {
      env: {
        // Database passthrough
        dbQuery: async (sqlPtr: number, sqlLen: number, paramsPtr: number, paramsLen: number): Promise<number> => {
          return 0; // Stub - will be implemented in Phase 1
        },
        dbExec: async (sqlPtr: number, sqlLen: number, paramsPtr: number, paramsLen: number): Promise<number> => {
          return 0; // Stub
        },
        cacheGet: async (keyPtr: number, keyLen: number): Promise<number> => {
          return 0; // Stub
        },
        cacheSet: (keyPtr: number, keyLen: number, valuePtr: number, valueLen: number, ttl: number): void => {
          // Stub
        },
        cacheDel: (keyPtr: number, keyLen: number): void => {
          // Stub
        },
      },
    });

    // Call the Go wasm init
    const exports = mod.instance.exports as any;
    if (exports._initialize) {
      exports._initialize();
    }

    memoryBuffer = exports.memory.buffer;

    // Call ready() to signal WASM is initialized
    if (exports.ready) {
      exports.ready();
    }

    const url = new URL(request.url);
    const headers: Record<string, string> = {};
    request.headers.forEach((value, key) => {
      headers[key] = value;
    });

    const body = request.body ? await request.text() : '';

    // Serialize request and call handleRequest via WASM
    const reqJSON = JSON.stringify({
      method: request.method,
      path: url.pathname,
      headers: JSON.stringify(headers),
      body: body,
      remote_addr: request.headers.get('cf-connecting-ip') || '127.0.0.1',
    });

    // TODO: Call WASM handleRequest and parse response
    // Placeholder response until WASM bridge is wired
    if (url.pathname === '/health' || url.pathname === '/api/v1/health') {
      return new Response(
        JSON.stringify({ status: 'ok', version: env.VERSION || '0.1.0', service: 'bendy-file-gateway' }),
        { headers: { 'content-type': 'application/json' } },
      );
    }

    // Serve admin dashboard for non-API routes
    if (!url.pathname.startsWith('/api/') && !url.pathname.startsWith('/admin/api/')) {
      return new Response('Bendy File Gateway - Admin dashboard will be served here', {
        headers: { 'content-type': 'text/html; charset=utf-8' },
      });
    }

    return new Response(
      JSON.stringify({ error: 'not_implemented', message: 'Gateway is initializing' }),
      { status: 501, headers: { 'content-type': 'application/json' } },
    );
  },
};
