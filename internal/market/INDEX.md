# INDEX.md

## Package

`internal/market`

## Role

日线行情领域、市场概览查询和本地 Provider 模式选择。

## Current Contents

- `domain/daily_bar.go`：Daily Bar 实体和最小领域校验。
- `domain/index_snapshot.go`：指数快照业务概念和校验。
- `domain/market_breadth.go`：市场宽度和成交额统计概念。
- `provider.go`：demo/real/fallback 模式及 Provider 名称解析；当前 real 仅为未来实现保留。
- `service.go`：`MarketOverview` 查询服务、四指数完整性校验和来源标记。
- `handler.go`：`GET /api/v1/markets/overview` HTTP 适配。
- `infrastructure/mysql_overview.go`：Seed/Provider 行情数据的 MySQL 查询与市场统计聚合。
