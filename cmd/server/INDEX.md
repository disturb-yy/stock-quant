# INDEX.md

## Package

`cmd/server`

## Role

程序启动入口与依赖组装。

## Typical Flow

```text
config
→ infrastructure clients
→ repositories/providers
→ application services
→ handlers
→ router
→ server
```

修改业务行为时不要优先改本 package。

## Current Contents

- `main.go`：从通用配置构造应用根日志实例，创建 HTTP Server 并监听 `:8080`。
- `router.go`：创建 Gin Router，按“请求日志 → Recovery”顺序安装中间件，创建 `/api/v1` 路由分组并装配 `internal/health` 的健康路由。
