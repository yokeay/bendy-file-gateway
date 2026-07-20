# Storage Driver Guide

## Driver Interface

所有存储驱动实现 `internal/storage/driver.go` 中定义的 `Driver` 接口：

```go
type Driver interface {
    Name() string
    Put(ctx context.Context, key string, body io.Reader, opts UploadOptions) (FileInfo, error)
    Get(ctx context.Context, key string, opts DownloadOptions) (io.ReadCloser, FileInfo, error)
    Head(ctx context.Context, key string) (FileInfo, error)
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, prefix string, limit int, continuationToken string) ([]FileInfo, string, error)
    Ping(ctx context.Context) error
}
```

## 驱动注册

使用 `init()` + `Register()` 模式：

```go
func init() {
    storage.Register("driver_name", func(cfg map[string]string) (storage.Driver, error) {
        // 验证必需配置
        // 返回 Driver 实例
    })
}
```

## 实现原则

1. **纯 HTTP 调用**：所有驱动使用标准库 `net/http`，不依赖第三方 SDK（避免 CGO）
2. **认证签名**：手动实现各云厂商的签名算法（如 AWS SigV4、阿里云签名）
3. **无状态**：驱动实例无状态，每次调用传入配置
4. **超时控制**：使用 `context.Context` 进行超时和取消控制

## 驱动配置格式

每个驱动的配置以 JSON 形式存储在 `backends.config` 字段中：

### S3 驱动
```json
{
  "endpoint": "https://s3.amazonaws.com",
  "region": "us-east-1",
  "bucket": "my-bucket",
  "access_key": "AKIA...",
  "secret_key": "...",
  "path_style": false
}
```

### 阿里云 OSS
```json
{
  "endpoint": "https://oss-cn-hangzhou.aliyuncs.com",
  "bucket": "my-bucket",
  "access_key_id": "LTAI...",
  "access_key_secret": "..."
}
```

### 华为云 OBS
```json
{
  "endpoint": "https://obs.cn-north-4.myhuaweicloud.com",
  "bucket": "my-bucket",
  "access_key": "...",
  "secret_key": "..."
}
```

### 七牛云 Kodo
```json
{
  "access_key": "...",
  "secret_key": "...",
  "bucket": "my-bucket",
  "region": "z0"
}
```

### 腾讯云 COS
```json
{
  "bucket": "my-bucket-1250000000",
  "region": "ap-guangzhou",
  "secret_id": "...",
  "secret_key": "..."
}
```

### 天翼云 OOS
```json
{
  "endpoint": "https://oos-cn.ctyunapi.cn",
  "bucket": "my-bucket",
  "access_key": "...",
  "secret_key": "..."
}
```

### 联通云 OSS
```json
{
  "endpoint": "https://oss.cucloud.cn",
  "bucket": "my-bucket",
  "access_key": "...",
  "secret_key": "..."
}
```

### Redis 存储
```json
{
  "host": "127.0.0.1",
  "port": "6379",
  "password": "",
  "db": "0",
  "prefix": "bgw:"
}
```

### PostgreSQL 存储
```json
{
  "dsn": "postgres://user:pass@host:5432/db?sslmode=require",
  "table": "storage_files",
  "key_column": "key",
  "data_column": "data"
}
```

### MySQL 存储
```json
{
  "dsn": "user:pass@tcp(host:3306)/db",
  "table": "storage_files",
  "key_column": "storage_key",
  "data_column": "file_data"
}
```

## 添加新驱动

1. 在 `internal/storage/drivers/` 下创建新文件（如 `newcloud.go`）
2. 实现 `Driver` 接口的所有方法
3. 在 `init()` 中调用 `storage.Register("driver_name", factory)`
4. 在 `cmd/gateway/main.go` 中添加空白导入 `_ "bendy-file-gateway/internal/storage/drivers"`
5. 更新本文档

## 签名参考

### AWS Signature V4（S3、阿里云 OSS 等兼容）

S3 兼容存储使用 AWS SigV4 签名：
1. 创建规范请求（Canonical Request）
2. 创建待签字符串（String to Sign）
3. 计算签名
4. 添加 Authorization 头

### 阿里云 OSS 签名（非 S3 兼容模式）

阿里云 OSS 支持两种模式：
- S3 兼容模式（推荐）：使用 AWS SigV4
- 原生模式：使用阿里云自有的签名算法

### 华为云 OBS 签名

华为云 OBS 兼容 AWS SigV4 签名。

### 七牛云 Kodo 签名

七牛云使用管理令牌（Access Token）：
- 上传令牌：基于上传策略生成
- 管理令牌：基于管理 URL 生成

### 腾讯云 COS 签名

腾讯云 COS 兼容 AWS SigV4 签名，也支持自有的签名算法。

### 天翼云 OOS 签名

天翼云 OOS 兼容 AWS SigV4 签名。

### 联通云 OSS 签名

联通云 OSS 兼容 S3 API。
