# bendy-file-gateway API Documentation

## Tenant API (`/api/v1`)

所有租户请求需要 `Authorization: HMAC-SHA256 AccessKey:Signature` 和 `X-Bendy-Timestamp` 头。

### 文件操作

#### 上传文件
```
POST /api/v1/files/upload
Content-Type: multipart/form-data
Authorization: HMAC-SHA256 AK123:abcdef...

Body:
  file: <binary>
  path: /virtual/path/filename.ext    (可选，默认 /)
  backend_id: uuid                     (可选，使用默认后端)

Response 201:
{
  "id": "uuid",
  "virtual_path": "/photos/sunset.jpg",
  "storage_key": "tenant-id/uuid.jpg",
  "size": 1048576,
  "content_type": "image/jpeg",
  "checksum": "sha256:abc123...",
  "created_at": "2026-07-20T10:00:00Z"
}
```

#### 下载文件
```
GET /api/v1/files/download?path=/virtual/path/file.ext
  或
GET /api/v1/files/:id/download
Authorization: HMAC-SHA256 AK123:signature

Response 200: <binary file content>
Headers: Content-Type, Content-Disposition, Content-Length, ETag
```

#### 获取文件信息
```
GET /api/v1/files/info?path=/virtual/path/file.ext
  或
GET /api/v1/files/:id
Authorization: HMAC-SHA256 AK123:signature

Response 200:
{
  "id": "uuid",
  "virtual_path": "/photos/sunset.jpg",
  "size": 1048576,
  "content_type": "image/jpeg",
  "checksum": "sha256:abc123...",
  "created_at": "2026-07-20T10:00:00Z",
  "updated_at": "2026-07-20T10:00:00Z"
}
```

#### 列出文件
```
GET /api/v1/files/list?path=/virtual/dir&page=1&page_size=50
Authorization: HMAC-SHA256 AK123:signature

Response 200:
{
  "items": [
    { "id": "...", "virtual_name": "file.txt", "size": 1024, "content_type": "text/plain" }
  ],
  "total": 100,
  "page": 1,
  "page_size": 50
}
```

#### 删除文件
```
DELETE /api/v1/files/:id
Authorization: HMAC-SHA256 AK123:signature

Response 204
```

### 目录操作

#### 创建目录
```
POST /api/v1/directories
Authorization: HMAC-SHA256 AK123:signature
Content-Type: application/json

{
  "parent_path": "/photos",
  "name": "2026"
}

Response 201:
{
  "id": "uuid",
  "path": "/photos/2026",
  "created_at": "2026-07-20T10:00:00Z"
}
```

#### 列出目录
```
GET /api/v1/directories?path=/photos
Authorization: HMAC-SHA256 AK123:signature

Response 200:
{
  "id": "uuid",
  "path": "/photos",
  "directories": [
    { "id": "...", "name": "2026", "created_at": "..." }
  ],
  "files": [
    { "id": "...", "virtual_name": "cover.jpg", "size": 2048 }
  ]
}
```

#### 删除目录
```
DELETE /api/v1/directories?path=/photos/2026
Authorization: HMAC-SHA256 AK123:signature

Response 204 (仅空目录可删除)
```

### 配额查询

```
GET /api/v1/quota
Authorization: HMAC-SHA256 AK123:signature

Response 200:
{
  "traffic_limit": 10737418240,
  "traffic_used": 524288000,
  "api_calls_limit": 100000,
  "api_calls_used": 1234,
  "expiry_at": "2026-12-31T23:59:59Z"
}
```

---

## Admin API (`/admin/api/v1`)

所有管理端请求需要 `session_token` Cookie（HttpOnly）。

### 认证

#### GitHub OAuth 登录
```
POST /admin/api/v1/auth/github
Content-Type: application/json

{ "code": "github_oauth_code" }

Response 200:
{
  "admin": {
    "id": "uuid",
    "github_username": "user1",
    "name": "User Name",
    "avatar_url": "https://...",
    "role": "admin"
  },
  "redirect": "/dashboard"
}

Set-Cookie: session_token=uuid; HttpOnly; Secure; SameSite=Lax; Path=/
```

#### 获取当前管理员
```
GET /admin/api/v1/auth/me

Response 200:
{
  "id": "uuid",
  "github_username": "user1",
  "name": "User Name",
  "avatar_url": "https://...",
  "role": "admin"
}
```

#### 退出登录
```
POST /admin/api/v1/auth/logout

Response 204
Clears session_token cookie
```

### 统计面板

```
GET /admin/api/v1/stats

Response 200:
{
  "total_tenants": 42,
  "active_tenants": 38,
  "total_files": 15000,
  "total_bytes": 10737418240,
  "api_calls_today": 50000,
  "api_calls_this_month": 1500000
}
```

### 租户管理

#### 列出租户
```
GET /admin/api/v1/tenants?page=1&page_size=20&status=active

Response 200:
{
  "items": [
    {
      "id": "uuid",
      "name": "My App",
      "access_key": "AKxxxxxxxxxxxx",
      "status": "active",
      "created_at": "2026-07-20T10:00:00Z"
    }
  ],
  "total": 42,
  "page": 1,
  "page_size": 20
}
```

#### 创建租户
```
POST /admin/api/v1/tenants
Content-Type: application/json

{
  "name": "My App",
  "traffic_limit": 10737418240,
  "api_calls_limit": 100000,
  "expiry_at": "2026-12-31T23:59:59Z"
}

Response 201:
{
  "tenant": { "id": "uuid", "name": "My App", "status": "active", ... },
  "access_key": "AKxxxxxxxxxxxx",
  "secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

#### 获取租户详情
```
GET /admin/api/v1/tenants/:id

Response 200: { tenant details, not including secret }
```

#### 更新租户
```
PATCH /admin/api/v1/tenants/:id
Content-Type: application/json

{
  "name": "New Name",
  "status": "suspended"
}

Response 200
```

#### 删除租户
```
DELETE /admin/api/v1/tenants/:id

Response 204
```

#### 轮换密钥
```
POST /admin/api/v1/tenants/:id/rotate-key

Response 201:
{
  "access_key": "AKyyyyyyyyyyyy",
  "secret": "yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy"
}
```

### 配额管理

#### 查看配额
```
GET /admin/api/v1/tenants/:id/quota

Response 200:
{
  "traffic_limit": 10737418240,
  "traffic_used": 524288000,
  "api_calls_limit": 100000,
  "api_calls_used": 1234,
  "expiry_at": "2026-12-31T23:59:59Z"
}
```

#### 更新配额
```
PATCH /admin/api/v1/tenants/:id/quota
Content-Type: application/json

{
  "traffic_limit": 21474836480,
  "api_calls_limit": 200000,
  "expiry_at": "2027-06-30T23:59:59Z"
}

Response 200
```

### 存储后端管理

#### 列出后端
```
GET /admin/api/v1/tenants/:id/backends

Response 200:
{
  "items": [
    { "id": "uuid", "name": "prod-s3", "driver": "s3", "status": "active" }
  ]
}
```

#### 创建后端
```
POST /admin/api/v1/tenants/:id/backends
Content-Type: application/json

{
  "driver": "s3",
  "name": "prod-s3",
  "is_default": true,
  "config": {
    "endpoint": "https://s3.amazonaws.com",
    "region": "us-east-1",
    "bucket": "my-bucket",
    "access_key": "...",
    "secret_key": "..."
  }
}

Response 201
```

#### 更新后端
```
PATCH /admin/api/v1/backends/:id
Content-Type: application/json

{ "config": { ... } }

Response 200
```

#### 删除后端
```
DELETE /admin/api/v1/backends/:id

Response 204
```

---

## 错误码

| HTTP 状态码 | 错误码 | 说明 |
|-------------|--------|------|
| 400 | `invalid_request` | 请求参数错误 |
| 401 | `unauthorized` | 认证失败 |
| 403 | `forbidden` | 权限不足 / 密钥过期 |
| 404 | `not_found` | 文件或目录不存在 |
| 409 | `conflict` | 目录非空 | 文件名冲突 |
| 413 | `payload_too_large` | 文件超过大小限制 |
| 429 | `quota_exceeded` | 配额超限 |
| 500 | `internal_error` | 服务器内部错误 |
| 502 | `backend_error` | 存储后端错误 |
| 503 | `service_unavailable` | 服务不可用 |

错误响应格式：
```json
{
  "error": {
    "code": "quota_exceeded",
    "message": "API calls limit exceeded",
    "details": {
      "limit": 100000,
      "used": 100000,
      "retry_after": "2026-08-01T00:00:00Z"
    }
  }
}
```
