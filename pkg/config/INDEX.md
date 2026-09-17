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

- `config.go`：读取服务标识、HTTP 监听地址、通用运行时日志和 Tushare 同步配置，并提供开发、生产环境的默认值。

## Application Environment Variables

| 变量 | 默认值 | 说明 |
|---|---|---|
| `SERVICE_NAME` | `stock-quant` | 应用根日志中的服务标识；部署时由 Helm values 注入。 |
| `HTTP_ADDRESS` | `:8357` | HTTP Server 监听地址；必须是合法的 `host:port` 且端口在 1-65535 之间。 |

## Logging Environment Variables

| 变量 | 默认值 | 说明 |
|---|---|---|
| `APP_ENV` | `development` | 值为 `production` 时默认使用 JSON 输出。 |
| `LOG_LEVEL` | `info` | `slog` 日志等级，例如 `debug`、`info`、`warn`、`error`。 |
| `LOG_FORMAT` | 随 `APP_ENV` 决定 | 显式设为 `text` 或 `json` 时覆盖环境默认值，大小写不敏感。 |
| `LOG_OUTPUT` | `console` | 日志输出位置，支持 `console` 或 `file`，大小写不敏感。 |
| `LOG_DIR` | `logs` | `LOG_OUTPUT=file` 时使用的日志目录；文件名为 `<SERVICE_NAME>.log`，目录必须可写。 |

## Database Environment Variables

| 变量 | 默认值 | 说明 |
|---|---|---|
| `DB_HOST` | `127.0.0.1` | MySQL 地址。 |
| `DB_PORT` | `3307` | 本地编排映射的 MySQL 端口。 |
| `DB_NAME` | `stock_quant_dev` | 本地数据库名。 |
| `DB_USER` | `stock_quant` | 本地数据库用户。 |
| `DB_PASSWORD` | `stock_quant_dev` | 本地数据库密码；部署时通过安全配置覆盖。 |
| `DATA_PROVIDER` | `demo` | `demo` 使用版本化 fixture；`real` 或 `tushare` 启动前同步 Tushare 并以 MySQL 读模型提供 API。 |

## Tushare Environment Variables

| 变量 | 默认值 | 说明 |
|---|---|---|
| `TUSHARE_TOKEN` | 空 | Tushare Pro Token；真实模式必填，不写入日志。 |
| `TUSHARE_ENDPOINT` | `https://api.tushare.pro` | Tushare Pro HTTP endpoint。 |
| `TUSHARE_START_DATE` | 结束日前 30 天 | 同步起始日期，格式 `YYYYMMDD`。 |
| `TUSHARE_END_DATE` | 当前 UTC 日期 | 同步结束日期，格式 `YYYYMMDD`。 |
| `TUSHARE_LOOKBACK_DAYS` | `30` | 未设置起始日期时的回溯天数。 |
| `TUSHARE_TIMEOUT_SECONDS` | `15` | 单次 Tushare 请求超时秒数。 |
