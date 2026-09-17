# INDEX.md

## Package

`internal/market`

## Role

日线行情领域、市场概览查询和本地 Provider 模式选择。

## Current Contents

- `domain/daily_bar.go`：Daily Bar 实体和最小领域校验。
- `domain/daily_basic.go`：daily basic 估值与规模快照及最小领域校验。
- `domain/index_snapshot.go`：指数快照业务概念和校验。
- `domain/market_breadth.go`：市场宽度和成交额统计概念。
- `domain/sector.go`：行业分类、行业成分及等权表现聚合规则。
- `provider.go`：demo/real/fallback 模式、Tushare Provider 名称和来源元数据解析。
- `service.go`：`MarketOverview` 查询服务、四指数完整性校验和来源标记。
- `sectors.go`：`MarketSectors` 查询服务、最新交易日校验和来源标记。
- `signals.go`：市场信号扫描 application service、参数规范化、历史完整性校验和来源标记。
- `rankings.go`：股票排行榜 application service、metric 规范化、稳定排序后的分页和来源标记。
- `bars.go`：股票研究型日线 application service、范围/复权/基准参数校验、MA 与相对收益编排。
- `handler.go`：`GET /api/v1/markets/overview`、`GET /api/v1/markets/sectors`、`GET /api/v1/markets/signals`、`GET /api/v1/markets/rankings`、`GET /api/v1/stocks/{symbol}/bars` HTTP 适配。
- `domain/chart.go`：OHLC 复权、MA5/MA20 和共同交易日基准收益计算。
- `domain/signal.go`：放量、突破、新高、强势四类信号规则。
- `infrastructure/mysql_overview.go`、`infrastructure/mysql_sectors.go`、`infrastructure/mysql_signals.go`、`infrastructure/mysql_rankings.go`、`infrastructure/mysql_bars.go`：Seed/Provider 行情数据的 MySQL 查询与市场统计、信号历史、排行榜聚合、股票日线及基准序列。
