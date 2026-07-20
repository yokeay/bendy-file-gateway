/// <reference types="@vercel/node" />

import type { VercelRequest, VercelResponse } from '@vercel/node';
import { readFileSync } from 'fs';
import { join } from 'path';

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

let wasmModule: WebAssembly.Module | null = null;
let wasmBuffer: Buffer | null = null;

function loadWasmBuffer(): Buffer {
  if (!wasmBuffer) {
    try {
      wasmBuffer = readFileSync(join(process.cwd(), 'platforms/vercel/wasm/gateway.wasm'));
    } catch {
      try {
        wasmBuffer = readFileSync(join(process.cwd(), 'wasm/gateway.wasm'));
      } catch {
        throw new Error('WASM binary not found at platforms/vercel/wasm/gateway.wasm');
      }
    }
  }
  return wasmBuffer;
}

export default async function handler(req: VercelRequest, res: VercelResponse) {
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, PATCH, DELETE, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization, X-Bendy-Timestamp');

  if (req.method === 'OPTIONS') {
    return res.status(204).end();
  }

  try {
    const buf = loadWasmBuffer();
    if (!wasmModule) {
      wasmModule = await WebAssembly.compile(buf);
    }

    // Helpers wired after instantiation (WASM binary defines its own memory)
    let readCString: (ptr: number, maxLen?: number) => string = () => '';
    let allocString: (str: string) => number = () => 0;

    const envMap: Record<string, string> = {
      GITHUB_CLIENT_ID: process.env.ADMIN_GITHUB_CLIENT_ID || '',
      GITHUB_CLIENT_SECRET: process.env.ADMIN_GITHUB_CLIENT_SECRET || '',
      ADMIN_GITHUB_USERNAMES: process.env.ADMIN_GITHUB_USERNAMES || '',
      GITHUB_REDIRECT_URI: process.env.ADMIN_GITHUB_REDIRECT_URI || '',
      SESSION_SECRET: process.env.SESSION_SECRET || '',
      VERSION: process.env.VERSION || '0.1.0',
    };

    // WASM binary defines and exports its own memory (does NOT import env.memory)
    const instance = await WebAssembly.instantiate(wasmModule, {
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

    const exports = instance.exports as unknown as WASMExports;
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

    const initFn = exports._initialize || exports._start;
    if (initFn) initFn();

    if (exports.ready() !== 1) {
      return res.status(503).json({ status: 'error', message: 'WASM not ready' });
    }

    const path = req.url || '/';
    const headers: Record<string, string> = {};
    for (const [key, value] of Object.entries(req.headers)) {
      if (typeof value === 'string') headers[key] = value;
      else if (Array.isArray(value)) headers[key] = value.join(', ');
    }
    const body = req.body ? (typeof req.body === 'string' ? req.body : JSON.stringify(req.body)) : '';
    const remoteAddr = (req.headers['x-forwarded-for'] as string) || req.socket?.remoteAddress || '127.0.0.1';

    const writeStr = (str: string): { ptr: number; len: number } => {
      const bytes = encoder.encode(str);
      const ptr = exports.malloc(bytes.length);
      new Uint8Array(mem.buffer, ptr, bytes.length).set(bytes);
      return { ptr, len: bytes.length };
    };

    const m = writeStr(req.method || 'GET');
    const p = writeStr(path);
    const h = writeStr(JSON.stringify(headers));
    const b = writeStr(body);
    const a = writeStr(remoteAddr);

    const resultPtr = exports.handleRequest(m.ptr, m.len, p.ptr, p.len, h.ptr, h.len, b.ptr, b.len, a.ptr, a.len);

    const resultJSON = readCString(Number(resultPtr));
    let result: { status_code?: number; headers?: Record<string, string>; body?: string };
    try {
      result = JSON.parse(resultJSON);
    } catch {
      return res.status(500).json({ error: 'invalid_wasm_response' });
    }

    if (result.headers) {
      for (const [key, value] of Object.entries(result.headers)) {
        res.setHeader(key, value as string);
      }
    }

    const statusCode = result.status_code || 200;
    if (result.body) {
      return res.status(statusCode).send(result.body);
    }
    return res.status(statusCode).end();
  } catch (err) {
    console.error('WASM gateway error:', err);
    return res.status(500).json({
      error: 'internal_error',
      message: 'Gateway encountered an error',
    });
  }
}
