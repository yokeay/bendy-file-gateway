# Development Phases & Technical Solutions

## Phase 0: 项目脚手架 (Scaffolding)

**目标**：空项目在两个平台上成功构建和部署。

### 技术方案
- **Go 模块初始化**：`go mod init github.com/bendy/file-gateway`
- **React 项目**：Vite + React 18 + TypeScript，`npm create vite@latest web -- --template react-ts`
- **WASM 编译**：TinyGo `wasm` target，输出 `gateway.wasm`
- **平台入口**：Cloudflare Worker + Vercel Function 加载 WASM 并返回 "Hello World"

### 产出
- `go.mod`, `cmd/gateway/main.go` (最小 WASM 出口)
- `platforms/cloudflare/index.ts`, `platforms/vercel/api/gateway.ts`
- `scripts/build-wasm.sh`, `scripts/build-web.sh`, `Makefile`
- Git 三分支：`main`, `cf`, `vercel`

---

## Phase 1: 核心框架 (Core Framework)

**目标**：WASM 桥接工作，请求路由工作，数据库 Schema 部署。

### 技术方案
- **WASM 桥接**：实现 `internal/wasm/exports.go` 和 `imports.go`，使用 TinyGo 的 `//go:wasmimport` 指令
- **路由**：自实现轻量路由器（`internal/server/router.go`），支持路径参数和方法匹配
- **中间件链**：Recovery → CORS → Logging → (Auth) → (Quota) → Router
- **DB 桥接**：通过 WASM import 调用 JS host 执行 SQL，Go 侧传递参数化 SQL + 参数 JSON
- **迁移**：创建 schema SQL 文件，JS host 在启动时执行

### 产出
- 完整的 `internal/wasm/` 包
- `internal/server/` 路由器 + 中间件框架
- `internal/db/bridge.go` + `internal/cache/bridge.go`
- `platforms/cloudflare/schema.sql` + `platforms/vercel/schema.sql`

### 关键技术决策
- **不使用 `net/http` ServeMux**：TinyGo WASM 不支持完整的 `net/http` 服务器，需要自实现请求处理
- **不使用 ORM**：SQL 直接编写，通过桥接层传递，保持最小依赖
- **ID 生成**：使用 `crypto/rand` 生成 UUID v4

---

## Phase 2: 认证 + 租户系统

**目标**：租户认证工作，多租户隔离验证。

### 技术方案
- **HMAC 签名**：`HMAC-SHA256(AccessKey, string-to-sign)`, string-to-sign = `METHOD\nPATH\nTIMESTAMP\nBODY_SHA256`
- **防重放**：检查 `X-Bendy-Timestamp` 在 ±5 分钟内
- **密钥存储**：Secret 使用 PBKDF2 或直接 SHA256 hash 存储
- **GitHub OAuth**：标准 OAuth 2.0 流程，验证 GitHub username 在 `ADMIN_GITHUB_USERNAMES` 列表中
- **Session**：UUID token，24h 过期，HttpOnly Cookie

### 产出
- `internal/auth/tenant.go` — HMAC 验证
- `internal/auth/github.go` — GitHub OAuth 客户端
- `internal/auth/session.go` — Session 管理
- `internal/handler/tenant.go` — 租户 CRUD

---

## Phase 3: 存储驱动 (S3 + Aliyun OSS)

**目标**：至少 2 个存储后端完全可用。

### 技术方案
- **驱动接口**：`internal/storage/driver.go`
- **驱动注册**：`internal/storage/registry.go`，使用 `init()` + `map[string]Factory`
- **多驱动管理**：`internal/storage/manager.go`，根据 backend_id 路由请求
- **S3 驱动**：纯 HTTP 请求，手动实现 AWS Signature V4
- **阿里云 OSS**：支持 S3 兼容模式（推荐）和原生模式

### 关键实现
- AWS SigV4 签名（含 chunked upload 支持）
- Content-Type 自动检测
- 大文件分块上传（后续迭代）

---

## Phase 4: 虚拟目录

**目标**：完整的虚拟文件系统，支持嵌套目录。

### 技术方案
- **目录树**：`directories` 表，`parent_id` 自引用
- **路径解析**：将 `/a/b/c/file.txt` 拆分为组件，逐级查找
- **文件关联**：`files.directory_id` → `directories.id`
- **storage_key**：不透明，格式 `{tenant_id}/{uuid}.{ext}`

### 关键实现
- 唯一性约束：`(tenant_id, parent_id, name)` UNIQUE
- 级联删除：应用层处理（D1 不支持外键级联）
- 递归查询：Go 代码循环查询（避免递归 CTE，兼容 SQLite）

---

## Phase 5: 配额系统

**目标**：流量和 API 调用配额强制执行。

### 技术方案
- **原子更新**：`UPDATE ... WHERE traffic_used + :bytes <= traffic_limit`，返回 0 行表示超限
- **缓存**：配额数据缓存到 KV/Redis，60s TTL
- **异步日志**：`api_logs` 写入使用缓冲批量插入

### 配额检查流程
```
请求 → 缓存 → 命中? → 检查限额
              ↓ 未命中
             DB查询 → 填充缓存 → 检查限额
                           ↓
                    返回 429 或继续
```

---

## Phase 6: 剩余驱动

**目标**：全部 10 个存储驱动实现完成。

### 驱动优先级
1. 腾讯云 COS（AWS SigV4 兼容）
2. 七牛云 Kodo（自有签名）
3. 华为云 OBS（SigV4 兼容）
4. 天翼云 OOS（SigV4 兼容）
5. 联通云 OSS（S3 兼容）
6. Redis 存储（List/Hash 模拟）
7. PostgreSQL 存储（大对象/bytea）
8. MySQL 存储（LONGBLOB）

### 测试策略
- S3 驱动用 MinIO 模拟
- 国内云驱动用真实测试账号
- Redis/PG/MySQL 用 Docker 本地测试

---

## Phase 7: 管理后台

**目标**：完整的管理后台，包含所有功能页面。

### 技术方案
- **构建**：Vite + React 18 + TypeScript
- **路由**：React Router v6，`BrowserRouter`
- **状态**：React Context + useReducer
- **UI**：CSS custom properties，macOS 风格设计系统
- **i18n**：自定义 hook，JSON 文件存储翻译

### 页面
| 页面 | 路由 | 功能 |
|------|------|------|
| 登录 | `/login` | GitHub OAuth 登录按钮 |
| 仪表盘 | `/` | 统计卡片、趋势图 |
| 文件 | `/files` | 文件浏览器，上传/下载 |
| 租户 | `/tenants` | CRUD 表格，创建/编辑弹窗 |
| 配额 | `/tenants/:id/quota` | 编辑限额，查看用量 |
| 设置 | `/settings` | 主题/语言切换，管理员列表 |

### macOS 设计系统
- **字体**：`-apple-system, BlinkMacSystemFont, "SF Pro Display"`
- **圆角**：8px (小), 12px (中), 16px (大)
- **阴影**：微妙扩散阴影，模拟 macOS 窗口
- **过渡**：200ms ease-out
- **毛玻璃**：`backdrop-filter: blur(20px)` 用于侧边栏和标题栏

---

## Phase 8: 加固

**目标**：生产就绪。

### 任务
- 异步日志写入（buffer + batch insert）
- 每租户速率限制（token bucket）
- 健康检查端点 `GET /health`
- 错误标准化（JSON 格式，code + message + details）
- 安全审计（签名验证、SQL 注入、路径遍历、SSRF）
- 文档完善
