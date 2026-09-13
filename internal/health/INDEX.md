# INDEX.md

## Package

`internal/health`

## Role

跨领域健康 HTTP Interface Adapter。

## Current Contents

- `handler.go`：注册 `GET /heartbeat` 存活心跳和 `GET /probe` 服务探针；当前仅检查进程能否响应。
- `handler_test.go`：验证两个健康接口的 HTTP 状态码和 JSON 响应。
