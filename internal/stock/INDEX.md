# INDEX.md

## Package

`internal/stock`

## Role

股票代码、名称、交易所和上市状态等基础信息，以及股票详情概览查询。

## Current Contents

- `domain/instrument.go`：Instrument 实体、状态枚举和最小领域校验。
- `overview.go`：股票详情概览响应、聚合读取边界、symbol 校验和应用服务。
- `financials.go`：股票财务报告查询、报告期范围选择、摘要/指标响应和来源语义。
- `valuation.go`：股票估值快照查询、历史分位、同日行业同行中位数和来源语义。
- `handler.go`：`GET /api/v1/stocks/{symbol}` HTTP 适配及统一错误映射。
- `financials_handler.go`：`GET /api/v1/stocks/{symbol}/financials` 参数解析及统一错误映射。
- `valuation_handler.go`：`GET /api/v1/stocks/{symbol}/valuation` 参数解析及统一错误映射。
- `infrastructure/mysql_overview.go`：身份、同日行情、daily basic、财务指标和 20 日 OHLC 点的 MySQL 聚合读取。
- `infrastructure/mysql_financials.go`：财务报告、三张简化报表字段和 Seed 来源的 MySQL 读取。
