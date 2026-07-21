const BASE = '/admin/api/v1';

async function request<T>(path: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...opts?.headers },
    ...opts,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || body.message || `HTTP ${res.status}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function get<T>(path: string): Promise<T> { return request<T>(path); }

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function post<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined });
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function put<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, { method: 'PUT', body: body ? JSON.stringify(body) : undefined });
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function del<T>(path: string): Promise<T> {
  return request<T>(path, { method: 'DELETE' });
}

export interface DashboardStats {
  total_tenants: number;
  total_files: number;
  traffic_used: number;
  storage_used: number;
  api_calls_today: number;
}

export interface Tenant {
  id: string;
  name: string;
  access_key: string;
  status: string;
  backend: string;
  backend_config: string;
  expires_at: string;
  created_at: string;
  updated_at: string;
}

export interface TenantQuota {
  id: string;
  tenant_id: string;
  tenant_name: string;
  traffic_limit: number;
  traffic_used: number;
  api_calls_limit: number;
  api_calls_used: number;
  storage_limit: number;
  storage_used: number;
  expires_at: string;
}

export interface FileInfo {
  key: string;
  size: number;
  content_type: string;
  last_modified: string;
}

export interface AdminUser {
  id: string;
  username: string;
  avatar_url: string;
  role: string;
}

export const api = {
  // Dashboard
  stats: () => get<DashboardStats>('/stats'),

  // Tenants
  listTenants: () => get<Tenant[]>('/tenants'),
  createTenant: (data: Partial<Tenant>) => post<Tenant>('/tenants', data),
  updateTenant: (id: string, data: Partial<Tenant>) => put<Tenant>(`/tenants/${id}`, data),
  deleteTenant: (id: string) => del<void>(`/tenants/${id}`),

  // Quotas
  listQuotas: () => get<TenantQuota[]>('/quotas'),
  updateQuota: (id: string, data: Partial<TenantQuota>) => put<TenantQuota>(`/quotas/${id}`, data),

  // Files
  listFiles: (prefix?: string) => get<{ files: FileInfo[]; next_token: string }>('/files' + (prefix ? `?prefix=${encodeURIComponent(prefix)}` : '')),
  deleteFile: (key: string) => del<void>(`/files/${encodeURIComponent(key)}`),

  // Auth
  me: () => get<{ user: AdminUser }>('/auth/me'),
  logout: () => post<void>('/auth/logout'),
};

// Format utilities
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i];
}

export function formatDate(s: string): string {
  if (!s) return '-';
  return new Date(s).toLocaleDateString();
}
