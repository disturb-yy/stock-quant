# INDEX.md

## Package

`internal/market`

## Role

日线行情领域、市场概览查询和本地 Provider 模式选择。

## Current Contents

- `domain/daily_bar.go`：Daily Bar 实体和最小领域校验。
- `domain/index_snapshot.go`：指数快照业务概念和校验。
- `domain/market_breadth.go`：市场宽度和成交额统计概念。
- `domain/sector.go`：行业分类、行业成分及等权表现聚合规则。
- `provider.go`：demo/real/fallback 模式及 Provider 名称解析；当前 real 仅为未来实现保留。
- `service.go`：`MarketOverview` 查询服务、四指数完整性校验和来源标记。
- `sectors.go`：`MarketSectors` 查询服务、最新交易日校验和来源标记。
- `handler.go`：`GET /api/v1/markets/overview`、`GET /api/v1/markets/sectors` HTTP 适配。
- `infrastructure/mysql_overview.go`、`infrastructure/mysql_sectors.go`：Seed/Provider 行情数据的 MySQL 查询与市场统计聚合。
