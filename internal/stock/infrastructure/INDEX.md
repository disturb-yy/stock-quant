# INDEX.md

## Package

`internal/stock/infrastructure`

## Role

聚合股票身份、同日最新行情、daily basic、财务指标和 20 日 OHLC 点。

## Current Contents

- `mysql_overview.go`：`GET /api/v1/stocks/{symbol}` 所需的 MySQL 聚合查询实现。
- `mysql_financials.go`：`GET /api/v1/stocks/{symbol}/financials` 所需的报告期、三张简化报表和来源查询实现。
- `mysql_valuation.go`：`GET /api/v1/stocks/{symbol}/valuation` 所需的估值历史、行业成员和来源查询实现。
