# INDEX.md

## Package

`internal/stock`

## Role

股票代码、名称、交易所和上市状态等基础信息，以及股票详情概览查询。

## Current Contents

- `domain/instrument.go`：Instrument 实体、状态枚举和最小领域校验。
- `overview.go`：股票详情概览响应、聚合读取边界、symbol 校验和应用服务。
- `handler.go`：`GET /api/v1/stocks/{symbol}` HTTP 适配及统一错误映射。
- `infrastructure/mysql_overview.go`：身份、同日行情、daily basic、财务指标和 20 日 OHLC 点的 MySQL 聚合读取。
