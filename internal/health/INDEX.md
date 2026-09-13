# INDEX.md

## Package

`internal/health`

## Role

跨领域健康 HTTP Interface Adapter。

## Current Contents

- `handler.go`：在 API v1 路由分组中注册 `GET /health`；当前仅检查进程能否响应。
- `handler_test.go`：验证健康接口的 HTTP 状态码和 JSON 响应。
