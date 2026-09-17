# INDEX.md

## Package

`internal/market/infrastructure`

## Role

读取 Seed/Provider 已落库的指数快照、日行情，并聚合市场宽度和成交额。

## Current Contents

- `mysql_overview.go`：市场概览 MySQL 查询实现，并承载共享的 MySQL 读取器。
- `mysql_sectors.go`：行业分类成分与最新交易日收盘价 MySQL 查询实现。
- `mysql_signals.go`：按最新交易日读取在市股票窗口日行情，供信号扫描使用。
- `mysql_rankings.go`：读取最新与前一交易日的在市股票日行情及换手率 daily basic，供排行榜查询使用。
- `mysql_bars.go`：股票日线、复权因子和可选 `000300.SH` 连续基准序列查询。
- `provider_tushare.go`：Tushare Pro HTTP 适配与股票基础信息、日线、每日指标、复权因子、市场指数的幂等 MySQL 同步。
