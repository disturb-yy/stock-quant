# INDEX.md

## Package

`pkg/config`

## Role

读取和解析项目配置。

## Typical Content

```text
config.go
loader.go
```

业务配置的语义解释应由对应业务模块负责。

## Current Contents

- `config.go`：读取服务标识和通用运行时日志配置，并提供开发、生产环境的默认值。

## Application Environment Variables

| 变量 | 默认值 | 说明 |
|---|---|---|
| `SERVICE_NAME` | `stock-quant` | 应用根日志中的服务标识；部署时由 Helm values 注入。 |

## Logging Environment Variables

| 变量 | 默认值 | 说明 |
|---|---|---|
| `APP_ENV` | `development` | 值为 `production` 时默认使用 JSON 输出。 |
| `LOG_LEVEL` | `info` | `slog` 日志等级，例如 `debug`、`info`、`warn`、`error`。 |
| `LOG_FORMAT` | 随 `APP_ENV` 决定 | 显式设为 `text` 或 `json` 时覆盖环境默认值，大小写不敏感。 |
| `LOG_OUTPUT` | `console` | 日志输出位置，支持 `console` 或 `file`，大小写不敏感。 |
| `LOG_DIR` | `logs` | `LOG_OUTPUT=file` 时使用的日志目录；文件名为 `<SERVICE_NAME>.log`，目录必须可写。 |
