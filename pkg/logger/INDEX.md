# INDEX.md

## Package

`pkg/logger`

## Role

项目通用日志能力。

## Typical Content

```text
logger.go
redactor.go
context.go
http.go
```

## Current Contents

- `logger.go`：创建带 `service`、`environment` 根字段的 `slog.Logger`。
- `output.go`：按 `console` 或 `file` 打开日志输出；文件按服务名追加写入配置目录。
- `redactor.go`：在输出前脱敏敏感日志字段。
- `context.go`：保存 request-scoped logger 和 request ID。
- `http.go`：Gin 请求日志 middleware；将 Request ID 和 request-scoped logger 写入标准 request context，并记录状态码、路由和耗时。
