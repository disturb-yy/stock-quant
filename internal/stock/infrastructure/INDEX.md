# INDEX.md

## Package

`internal/stock/infrastructure`

## Role

聚合股票身份、同日最新行情、daily basic、财务指标和 20 日 OHLC 点。

## Current Contents

- `mysql_overview.go`：`GET /api/v1/stocks/{symbol}` 所需的 MySQL 聚合查询实现。
