import { readFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';
import './wasm_exec.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const wasmPath = join(__dirname, '..', 'platforms', 'wasm', 'gateway.wasm');

const encoder = new TextEncoder();
const decoder = new TextDecoder();

const wasmBuffer = readFileSync(wasmPath);

// In-memory mock database
const db = {
  tenants: new Map(),
  admins: new Map(),
  admin_sessions: new Map(),
  tenant_quotas: new Map(),
  backends: new Map(),
  api_logs: new Map(),
};

function dbExec(sql, params) {
  if (sql.includes('INSERT INTO tenants')) {
    const [id, name, accessKey, secretKey, status, createdAt, updatedAt] = params;
    db.tenants.set(id, { id, name, access_key: accessKey, secret_key_hash: secretKey, status, created_at: createdAt, updated_at: updatedAt });
    return { rows_affected: 1, last_insert_id: 0 };
  }
  if (sql.includes('INSERT INTO tenant_quotas')) {
    const [id, tenantID, tl, tu, acl, acu, sl, su, ca, ua] = params;
    db.tenant_quotas.set(tenantID, { id, tenant_id: tenantID, traffic_limit: tl, traffic_used: tu, api_calls_limit: acl, api_calls_used: acu, storage_limit: sl, storage_used: su, expiry_at: null, created_at: ca, updated_at: ua });
    return { rows_affected: 1, last_insert_id: 0 };
  }
  if (sql.includes('INSERT INTO admins')) {
    const [id, ghUser, ghId, name, avatar, role, lastLogin, ca, ua] = params;
    db.admins.set(id, { id, github_username: ghUser, github_id: ghId, name, avatar_url: avatar, role, last_login_at: lastLogin, created_at: ca, updated_at: ua });
    return { rows_affected: 1, last_insert_id: 0 };
  }
  if (sql.includes('INSERT INTO admin_sessions')) {
    const [id, adminId, sessionToken, expiresAt, createdAt] = params;
    db.admin_sessions.set(sessionToken, { id, admin_id: adminId, session_token: sessionToken, expires_at: expiresAt, created_at: createdAt });
    return { rows_affected: 1, last_insert_id: 0 };
  }
  if (sql.includes('INSERT INTO backends')) {
    const [id, tenantID, name, driver, config, isDefault, status, ca, ua] = params;
    db.backends.set(id, { id, tenant_id: tenantID, name, driver, config, is_default: isDefault, status, created_at: ca, updated_at: ua });
    return { rows_affected: 1, last_insert_id: 0 };
  }
  if (sql.includes('DELETE FROM tenants')) {
    const id = params[0];
    db.tenants.delete(id);
    return { rows_affected: 1, last_insert_id: 0 };
  }
  if (sql.includes('DELETE FROM admin_sessions')) {
    const token = params[0];
    db.admin_sessions.delete(token);
    return { rows_affected: 1, last_insert_id: 0 };
  }
  if (sql.includes('DELETE FROM backends')) {
    db.backends.delete(params[0]);
    return { rows_affected: 1, last_insert_id: 0 };
  }
  if (sql.includes('UPDATE tenants SET')) {
    const tenant = db.tenants.get(params[params.length - 1]);
    if (tenant) {
      if (sql.includes('name = ?')) tenant.name = params[0];
      if (sql.includes('status = ?')) tenant.status = params[0];
      if (sql.includes('secret_key_hash = ?')) tenant.secret_key_hash = params[0];
      tenant.updated_at = params[params.length - 2];
    }
    return { rows_affected: 1, last_insert_id: 0 };
  }
  if (sql.includes('UPDATE admins SET')) {
    const adminId = params[params.length - 1];
    const admin = db.admins.get(adminId);
    if (admin) {
      admin.name = params[0];
      admin.avatar_url = params[1];
      admin.last_login_at = params[2];
      admin.updated_at = params[3];
    }
    return { rows_affected: 1, last_insert_id: 0 };
  }
  if (sql.includes('UPDATE tenant_quotas SET')) {
    const tenantId = params[params.length - 1];
    const quota = db.tenant_quotas.get(tenantId);
    if (quota) {
      // Update the fields based on SQL
      if (sql.includes('traffic_limit = ?')) quota.traffic_limit = params[0];
      if (sql.includes('api_calls_limit = ?')) quota.api_calls_limit = params[0];
      if (sql.includes('storage_limit = ?')) quota.storage_limit = params[0];
      if (sql.includes('expiry_at = ?')) quota.expiry_at = params[0];
      quota.updated_at = params[params.length - 2];
    }
    return { rows_affected: 1, last_insert_id: 0 };
  }
  if (sql.includes('UPDATE backends SET')) {
    const backendId = params[params.length - 1];
    const backend = db.backends.get(backendId);
    if (backend) {
      if (sql.includes('name = ?')) backend.name = params[0];
      if (sql.includes('config = ?')) backend.config = params[0];
      if (sql.includes('status = ?')) backend.status = params[0];
      if (sql.includes('is_default = ?')) backend.is_default = params[0];
      backend.updated_at = params[params.length - 2];
    }
    return { rows_affected: 1, last_insert_id: 0 };
  }
  console.log('[DBEXEC]', sql.substring(0, 80), '| params:', JSON.stringify(params));
  return { rows_affected: 1, last_insert_id: 0 };
}

function dbQuery(sql, paramsArray) {
  const params = paramsArray || [];

  if (sql.includes('FROM tenants')) {
    if (sql.includes('ORDER BY created_at DESC')) {
      const result = Array.from(db.tenants.values())
        .sort((a, b) => b.created_at.localeCompare(a.created_at));
      return result;
    }
    if (sql.includes('WHERE id = ?')) {
      return [db.tenants.get(params[0])].filter(Boolean);
    }
    if (sql.includes('WHERE access_key = ?')) {
      return Array.from(db.tenants.values()).filter(t => t.access_key === params[0]);
    }
  }

  if (sql.includes('COUNT(*)') && sql.includes('tenants')) {
    return [{ count: db.tenants.size }];
  }
  if (sql.includes('COUNT(*)') && sql.includes('files')) {
    return [{ count: 0 }];
  }
  if (sql.includes('SUM(traffic_bytes)') && sql.includes('api_logs')) {
    return [{ total: 0 }];
  }
  if (sql.includes('SUM(storage_used)') && sql.includes('tenant_quotas')) {
    return [{ total: 0 }];
  }

  if (sql.includes('FROM admins')) {
    if (sql.includes('WHERE github_id = ?')) {
      return Array.from(db.admins.values()).filter(a => a.github_id === params[0]);
    }
    if (sql.includes('WHERE id = ?')) {
      return [db.admins.get(params[0])].filter(Boolean);
    }
  }

  if (sql.includes('FROM admin_sessions')) {
    if (sql.includes('s.id = ?')) {
      const session = db.admin_sessions.get(params[0]);
      return session ? [{ admin_id: session.admin_id, expires_at: session.expires_at }] : [];
    }
  }

  if (sql.includes('FROM tenant_quotas')) {
    if (sql.includes('WHERE tenant_id = ?')) {
      return [db.tenant_quotas.get(params[0])].filter(Boolean);
    }
  }

  if (sql.includes('FROM backends')) {
    if (sql.includes('WHERE tenant_id = ?')) {
      return Array.from(db.backends.values()).filter(b => b.tenant_id === params[0]);
    }
  }

  console.log('[DBQUERY]', sql.substring(0, 80));
  return [];
}

const go = new globalThis.Go();
Object.assign(go.importObject.env, {
  dbQuery: (sqlPtr, sqlLen, paramsPtr, paramsLen) => {
    const sql = readWasmStr(sqlPtr, sqlLen);
    const paramsJSON = readWasmStr(paramsPtr, paramsLen);
    const params = paramsJSON ? JSON.parse(paramsJSON) : [];
    const result = dbQuery(sql, params);
    return allocResult(JSON.stringify(result));
  },
  dbExec: (sqlPtr, sqlLen, paramsPtr, paramsLen) => {
    const sql = readWasmStr(sqlPtr, sqlLen);
    const paramsJSON = readWasmStr(paramsPtr, paramsLen);
    const params = paramsJSON ? JSON.parse(paramsJSON) : [];
    const result = dbExec(sql, params);
    return allocResult(JSON.stringify(result));
  },
  cacheGet: () => BigInt(0),
  cacheSet: () => {},
  cacheDel: () => {},
  envGet: (keyPtr, keyLen) => {
    const key = readWasmStr(keyPtr, keyLen);
    if (key === 'VERSION') return allocResult('0.1.0');
    if (key === 'ADMIN_GITHUB_USERNAMES') return allocResult('testadmin');
    return BigInt(0);
  },
  fetch: (mPtr, mLen, uPtr, uLen, hPtr, hLen, bPtr, bLen) => {
    return allocResult(JSON.stringify({
      status_code: 500,
      headers: {},
      body: 'fetch not available in test',
    }));
  },
});

const wasmModule = await WebAssembly.compile(wasmBuffer);
const instance = await WebAssembly.instantiate(wasmModule, go.importObject);
const exports = instance.exports;
const memory = exports.memory;

// Helpers for WASM memory access
function readWasmStr(ptr, len) {
  const numPtr = typeof ptr === 'bigint' ? Number(ptr) : ptr;
  if (numPtr === 0) return '';
  const bytes = new Uint8Array(memory.buffer, numPtr, len || 4096);
  let end = 0;
  while (end < (len || 4096) && bytes[end] !== 0) end++;
  return decoder.decode(bytes.subarray(0, end));
}

function allocResult(str) {
  if (!str) return BigInt(0);
  const bytes = encoder.encode(str + '\0');
  const ptr = exports.malloc(bytes.length);
  new Uint8Array(memory.buffer, ptr, bytes.length).set(bytes);
  return BigInt(ptr);
}

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

  const raw = readWasmStr(Number(resultPtr), 8192);
  try {
    return JSON.parse(raw);
  } catch (e) {
    return { error: 'parse_error', raw };
  }
}

// Helper: create a session and call as admin
function callAsAdmin(method, path, body) {
  // First, create an admin and session
  const adminId = 'admin-test-001';
  db.admins.set(adminId, {
    id: adminId,
    github_username: 'testadmin',
    github_id: 12345,
    name: 'Test Admin',
    avatar_url: '',
    role: 'admin',
    last_login_at: new Date().toISOString(),
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  });
  const sessionToken = 'test-session-token';
  db.admin_sessions.set(sessionToken, {
    id: sessionToken,
    admin_id: adminId,
    session_token: sessionToken,
    expires_at: new Date(Date.now() + 86400000).toISOString(),
    created_at: new Date().toISOString(),
  });

  return callHandleRequest(method, path, {
    cookie: `session_token=${sessionToken}`,
  }, body, '127.0.0.1');
}

// Test 1: Health check
console.log('\n=== Test: GET /health ===');
const r1 = callHandleRequest('GET', '/health', {}, '', '127.0.0.1');
console.log('Status:', r1.status_code, '| Body:', r1.body);

// Test 2: GitHub OAuth login with pre-resolved github_user
console.log('\n=== Test: POST /admin/api/v1/auth/github (pre-resolved) ===');
const r2 = callHandleRequest('POST', '/admin/api/v1/auth/github', {}, JSON.stringify({
  github_user: { id: 12345, login: 'testadmin', name: 'Test Admin', avatar_url: '' },
}), '127.0.0.1');
console.log('Status:', r2.status_code);
console.log('Body:', JSON.stringify(r2).substring(0, 200));

// Test 3: Create tenant (as admin)
console.log('\n=== Test: POST /admin/api/v1/tenants (create) ===');
const r3 = callAsAdmin('POST', '/admin/api/v1/tenants', JSON.stringify({ name: 'MyTenant' }));
console.log('Status:', r3.status_code);
console.log('Body:', JSON.stringify(r3).substring(0, 300));

// Extract tenant ID for later tests
let tenantId = null;
if (r3.body) {
  const parsed = JSON.parse(r3.body);
  tenantId = parsed.tenant?.id;
}
console.log('Tenant ID:', tenantId);

// Test 4: List tenants (as admin)
console.log('\n=== Test: GET /admin/api/v1/tenants (list) ===');
const r4 = callAsAdmin('GET', '/admin/api/v1/tenants', '');
console.log('Status:', r4.status_code);
console.log('Body:', JSON.stringify(r4).substring(0, 300));

let r5, r6, r7, r8;

// Test 5: Get tenant detail
if (tenantId) {
  console.log('\n=== Test: GET /admin/api/v1/tenants/detail ===');
  r5 = callAsAdmin('GET', '/admin/api/v1/tenants/detail?id=' + tenantId, '');
  console.log('Status:', r5.status_code);
  console.log('Body:', JSON.stringify(r5).substring(0, 300));
}

// Test 6: Get tenant quota
if (tenantId) {
  console.log('\n=== Test: GET /admin/api/v1/tenants/quota ===');
  r6 = callAsAdmin('GET', '/admin/api/v1/tenants/quota?tenant_id=' + tenantId, '');
  console.log('Status:', r6.status_code);
  console.log('Body:', JSON.stringify(r6).substring(0, 300));
}

// Test 7: Update tenant quota
if (tenantId) {
  console.log('\n=== Test: PATCH /admin/api/v1/tenants/quota ===');
  r7 = callAsAdmin('PATCH', '/admin/api/v1/tenants/quota?tenant_id=' + tenantId, JSON.stringify({
    traffic_limit: 1073741824,
    storage_limit: 5368709120,
  }));
  console.log('Status:', r7.status_code);
}

// Test 8: Create backend
if (tenantId) {
  console.log('\n=== Test: POST /admin/api/v1/tenants/backends (create) ===');
  r8 = callAsAdmin('POST', '/admin/api/v1/tenants/backends', JSON.stringify({
    tenant_id: tenantId,
    name: 'My S3 Bucket',
    driver: 's3',
    config: { bucket: 'my-bucket', region: 'us-east-1' },
    is_default: true,
  }));
  console.log('Status:', r8.status_code);
  console.log('Body:', JSON.stringify(r8).substring(0, 300));
}

// Test 9: Tenant API (no auth) - should still be 401
console.log('\n=== Test: POST /api/v1/files/upload (no auth) ===');
const r9 = callHandleRequest('POST', '/api/v1/files/upload', {}, '', '127.0.0.1');
console.log('Status:', r9.status_code);

// Test 10: Admin auth required (no cookie)
console.log('\n=== Test: GET /admin/api/v1/tenants (no auth) ===');
const r10 = callHandleRequest('GET', '/admin/api/v1/tenants', {}, '', '127.0.0.1');
console.log('Status:', r10.status_code);

// Summary
console.log('\n=== SUMMARY ===');
const tests = [
  ['GET /health', r1.status_code === 200],
  ['POST /admin/auth/github', r2.status_code === 200],
  ['POST /admin/tenants (create)', r3.status_code === 201],
  ['GET /admin/tenants (list)', r4.status_code === 200],
  ['GET /admin/tenants/detail', r5?.status_code === 200],
  ['GET /admin/tenants/quota', r6?.status_code === 200],
  ['PATCH /admin/tenants/quota', r7?.status_code === 200],
  ['POST /admin/tenants/backends', r8?.status_code === 201],
  ['POST /api/files/upload (no auth)', r9.status_code === 401],
  ['GET /admin/tenants (no auth)', r10.status_code === 401],
];
for (const [name, pass] of tests) {
  console.log(pass ? '  PASS' : '  FAIL', name);
}
