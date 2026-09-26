# internal/data 导航

## 目录职责

A 股数据 bounded context 的业务实现。

## 当前子目录

- adapter/：A 股数据特有的技术适配器。
- application/：数据同步和股票查询用例编排。
- config/：数据源和同步策略配置。
- domain/：数据同步、股票数据和查询领域模型。
- interface/：HTTP 边界适配。
