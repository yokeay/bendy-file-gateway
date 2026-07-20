# bendy-file-gateway Requirements Document

## 1. 项目概述

bendy-file-gateway 是一个统一的多云存储网关，为多种第三方存储后端提供单一的 API 接口。支持在 Vercel 和 Cloudflare Workers 两个平台上部署，后端使用 Go 编译为 WASM，前端使用 React 构建管理后台。

## 2. 部署要求

- 三个分支：`main`（共享代码）、`vercel`、`cf`
- Go 后端通过 TinyGo 编译为 WASM
- Cloudflare：Workers + D1（关系型数据库）+ KV（缓存）
- Vercel：WASM Function + PostgreSQL + Redis

## 3. 功能需求

### 3.1 统一文件 API
- 文件上传（支持任意格式）
- 文件下载（支持断点续传/Range）
- 文件查询（元数据、大小、类型、校验和）
- 虚拟目录（目录结构存储在关系型数据库）

### 3.2 多租户系统
- 租户隔离：AccessKey + Secret 认证
- 流量限制：总传输字节数上限
- 调用次数限制：API 调用次数上限
- 有效期限制：Key 过期时间
- 所有限额存储在关系型数据库中

### 3.3 存储后端（一期）
| 序号 | 驱动 | 标识符 |
|------|------|--------|
| 1 | 通用 S3 兼容存储 | `s3` |
| 2 | Redis 存储 | `redis` |
| 3 | PostgreSQL 存储 | `postgres` |
| 4 | MySQL 存储 | `mysql` |
| 5 | 阿里云 OSS | `aliyun_oss` |
| 6 | 华为云 OBS | `huawei_obs` |
| 7 | 七牛云 Kodo | `qiniu_kodo` |
| 8 | 腾讯云 COS | `tencent_cos` |
| 9 | 天翼云 OOS | `tianyi_oos` |
| 10 | 联通云 OSS | `unicom_oss` |

### 3.4 管理后台
- GitHub OAuth 登录（管理员账户配置在环境变量中）
- macOS 设计风格：黑白灰三色体系
- 深色/浅色模式切换
- 中英文切换
- 功能页面：仪表盘、文件管理、租户管理、配额管理、系统设置

## 4. 技术约束

- Go 编译为 WASM（TinyGo），不允许 CGO
- 最小化三方依赖
- 存储驱动优先使用 HTTP API 直接对接，避免引入 SDK
- 如果 Go 生态有成熟的开源存储 SDK 可考虑使用
- 必须可扩展，方便后续添加更多存储后端

## 5. 非功能需求

- 安全：HMAC 签名认证，常量时间比较，防重放攻击
- 性能：配额数据缓存，减少 DB 查询
- 可扩展：驱动注册模式，新增驱动只需添加单个文件
- 可移植：D1（SQLite 子集）和 PostgreSQL 双兼容 Schema
