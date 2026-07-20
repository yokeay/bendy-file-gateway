import { readFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import './wasm_exec.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const wasmPath = join(__dirname, '..', 'platforms', 'wasm', 'gateway.wasm');

const encoder = new TextEncoder();
const decoder = new TextDecoder();

const wasmBuffer = readFileSync(wasmPath);

const go = new globalThis.Go();
Object.assign(go.importObject.env, {
  dbQuery: () => BigInt(0),
  dbExec: () => BigInt(0),
  cacheGet: () => BigInt(0),
  cacheSet: () => {},
  cacheDel: () => {},
  envGet: () => BigInt(0),
  fetch: () => BigInt(0),
});

const wasmModule = await WebAssembly.compile(wasmBuffer);
const instance = await WebAssembly.instantiate(wasmModule, go.importObject);
const exports = instance.exports;
const memory = exports.memory;

go.run(instance);
await new Promise(r => setTimeout(r, 500));

console.log('=== WASM Ready:', exports.ready(), '===');

function callHandleRequest(method, path, headers, body, remoteAddr) {
  const m = encoder.encode(method);
  const p = encoder.encode(path);
  const h = encoder.encode(JSON.stringify(headers));
  const b = encoder.encode(body);
  const a = encoder.encode(remoteAddr);

  const mPtr = exports.malloc(m.length);
  new Uint8Array(memory.buffer, mPtr, m.length).set(m);
  const pPtr = exports.malloc(p.length);
  new Uint8Array(memory.buffer, pPtr, p.length).set(p);
  const hPtr = exports.malloc(h.length);
  new Uint8Array(memory.buffer, hPtr, h.length).set(h);
  const bPtr = exports.malloc(b.length);
  new Uint8Array(memory.buffer, bPtr, b.length).set(b);
  const aPtr = exports.malloc(a.length);
  new Uint8Array(memory.buffer, aPtr, a.length).set(a);

  const resultPtr = exports.handleRequest(
    mPtr, m.length,
    pPtr, p.length,
    hPtr, h.length,
    bPtr, b.length,
    aPtr, a.length
  );

  function readCString(ptr, maxLen) {
    const numPtr = typeof ptr === 'bigint' ? Number(ptr) : ptr;
    if (numPtr === 0) return '';
    const bytes = new Uint8Array(memory.buffer, numPtr, maxLen || 4096);
    let end = 0;
    while (end < (maxLen || 4096) && bytes[end] !== 0) end++;
    return decoder.decode(bytes.subarray(0, end));
  }

  const raw = readCString(Number(resultPtr), 4096);
  try {
    return JSON.parse(raw);
  } catch (e) {
    return { error: 'parse_error', raw };
  }
}

// Test 1: GET /health
console.log('\n=== Test: GET /health ===');
const healthResult = callHandleRequest('GET', '/health', {}, '', '127.0.0.1');
console.log('Status:', healthResult.status_code);
console.log('Body:', healthResult.body);

// Test 2: GET /unknown (404)
console.log('\n=== Test: GET /unknown ===');
const notFound = callHandleRequest('GET', '/unknown', {}, '', '127.0.0.1');
console.log('Status:', notFound.status_code);
console.log('Body:', notFound.body);

// Test 3: POST /api/v1/files/upload (no auth - should fail)
console.log('\n=== Test: POST /api/v1/files/upload ===');
const upload = callHandleRequest('POST', '/api/v1/files/upload', {}, '', '127.0.0.1');
console.log('Status:', upload.status_code);
console.log('Body:', upload.body);
