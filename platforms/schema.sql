-- bendy-file-gateway Database Schema
-- Compatible with: Cloudflare D1 and PostgreSQL
-- All PKs are TEXT UUIDs, timestamps are ISO 8601 strings

CREATE TABLE IF NOT EXISTS tenants (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    access_key TEXT NOT NULL UNIQUE,
    secret_key_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',  -- active | suspended | expired
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tenant_quotas (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    traffic_limit INTEGER NOT NULL DEFAULT 0,      -- bytes, 0 = unlimited
    traffic_used INTEGER NOT NULL DEFAULT 0,
    api_calls_limit INTEGER NOT NULL DEFAULT 0,     -- 0 = unlimited
    api_calls_used INTEGER NOT NULL DEFAULT 0,
    storage_limit INTEGER NOT NULL DEFAULT 0,       -- bytes, 0 = unlimited
    storage_used INTEGER NOT NULL DEFAULT 0,
    expiry_at TEXT,                                  -- ISO 8601, NULL = no expiry
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS backends (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    driver TEXT NOT NULL,      -- s3, redis, postgres, etc.
    config TEXT NOT NULL,      -- JSON blob with driver-specific config
    is_default INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS directories (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    parent_id TEXT REFERENCES directories(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    path TEXT NOT NULL,        -- full path from root, e.g. /photos/2024
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(tenant_id, parent_id, name)
);

CREATE TABLE IF NOT EXISTS files (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    directory_id TEXT REFERENCES directories(id) ON DELETE SET NULL,
    backend_id TEXT NOT NULL REFERENCES backends(id),
    filename TEXT NOT NULL,
    storage_key TEXT NOT NULL,  -- key used in the storage backend
    size INTEGER NOT NULL DEFAULT 0,
    content_type TEXT NOT NULL DEFAULT 'application/octet-stream',
    etag TEXT,
    metadata TEXT,              -- JSON blob for custom metadata
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS admin_sessions (
    id TEXT PRIMARY KEY,
    admin_id TEXT NOT NULL,
    session_token TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS api_logs (
    id TEXT PRIMARY KEY,
    tenant_id TEXT REFERENCES tenants(id) ON DELETE SET NULL,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    traffic_bytes INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    remote_addr TEXT,
    created_at TEXT NOT NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_tenants_access_key ON tenants(access_key);
CREATE INDEX IF NOT EXISTS idx_tenant_quotas_tenant ON tenant_quotas(tenant_id);
CREATE INDEX IF NOT EXISTS idx_backends_tenant ON backends(tenant_id);
CREATE INDEX IF NOT EXISTS idx_directories_tenant_parent ON directories(tenant_id, parent_id);
CREATE INDEX IF NOT EXISTS idx_directories_path ON directories(tenant_id, path);
CREATE INDEX IF NOT EXISTS idx_files_tenant ON files(tenant_id);
CREATE INDEX IF NOT EXISTS idx_files_directory ON files(directory_id);
CREATE INDEX IF NOT EXISTS idx_files_backend ON files(backend_id);
CREATE INDEX IF NOT EXISTS idx_admin_sessions_token ON admin_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_api_logs_tenant ON api_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_logs_created ON api_logs(created_at);
