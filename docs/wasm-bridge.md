# WASM Bridge Design

## 概述

Go 后端通过 TinyGo 编译为 WASM 模块，运行在 JS 宿主环境中。Go 代码不直接访问数据库或缓存，而是通过 WASM imports 调用 JS 宿主提供的函数。

## 编译

```bash
tinygo build -o gateway.wasm -target wasm -no-debug -opt=2 ./cmd/gateway
```

TinyGo 版本要求：>= 0.30.0

## Exports（Go → JS）

Go 向 JS 暴露的函数：

### handleRequest
```
go:export handleRequest
func handleRequest(
    methodPtr unsafe.Pointer,
    methodLen int32,
    pathPtr unsafe.Pointer,
    pathLen int32,
    headersPtr unsafe.Pointer,
    headersLen int32,
    bodyPtr unsafe.Pointer,
    bodyLen int32,
    remoteAddrPtr unsafe.Pointer,
    remoteAddrLen int32,
) int64
```

返回一个指针，指向包含 statusCode、headers 和 body 的序列化结构。

### ready
```
go:export ready
func ready() int32
```

返回 1 表示 WASM 模块已初始化完成。

## Imports（JS → Go）

JS 宿主提供给 Go 的函数：

### dbQuery
```go
//go:wasmimport env dbQuery
func dbQuery(sqlPtr unsafe.Pointer, sqlLen int32, paramsPtr unsafe.Pointer, paramsLen int32) int64
```

执行查询 SQL，返回 JSON 格式的行数据。

### dbExec
```go
//go:wasmimport env dbExec
func dbExec(sqlPtr unsafe.Pointer, sqlLen int32, paramsPtr unsafe.Pointer, paramsLen int32) int64
```

执行变更 SQL（INSERT/UPDATE/DELETE），返回 `{ rows_affected, last_insert_id }` JSON。

### cacheGet
```go
//go:wasmimport env cacheGet
func cacheGet(keyPtr unsafe.Pointer, keyLen int32) int64
```

从缓存获取值。返回指针，Go 侧解析后判断是否 found。

### cacheSet
```go
//go:wasmimport env cacheSet
func cacheSet(keyPtr unsafe.Pointer, keyLen int32, valuePtr unsafe.Pointer, valueLen int32, ttlSeconds int32)
```

设置缓存值并指定 TTL。

### cacheDel
```go
//go:wasmimport env cacheDel
func cacheDel(keyPtr unsafe.Pointer, keyLen int32)
```

删除缓存项。

## 内存管理

Go 和 JS 之间的内存通过共享线性内存传递。字符串和字节数组需要手动管理：

1. Go 分配内存 → 返回指针给 JS
2. JS 读取内存 → 调用 Go 函数释放（防止泄漏）
3. JS 传入数据 → 先复制到 Go 内存 → 传递指针

使用自定义的 `malloc`/`free` 或 TinyGo 的 `export malloc`/`export free`。

## 平台实现差异

### Cloudflare Workers

```typescript
// platforms/cloudflare/bindings/db.ts
export async function dbQuery(sql: string, params: string): Promise<string> {
  const { results } = await env.DB.prepare(sql).bind(...JSON.parse(params)).all();
  return JSON.stringify(results);
}
```

### Vercel

```typescript
// platforms/vercel/bindings/db.ts
import { sql as pgSql } from '@vercel/postgres';

export async function dbQuery(sql: string, params: string): Promise<string> {
  const { rows } = await pgSql.query(sql, JSON.parse(params));
  return JSON.stringify(rows);
}
```

## 安全注意事项

- SQL 使用参数化查询，防止 SQL 注入
- 缓存 key 限制为特定前缀，防止 key 冲突
- WASM 内存访问边界检查
- 错误处理：所有 host 函数调用都需要 Go 侧错误处理
