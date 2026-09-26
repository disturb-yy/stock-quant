# internal/data 局部规约

本目录是 A 股数据 bounded context。遵循父级 internal/AGENTS.md 与根 AGENTS.md。

## 职责

- 管理 Instrument、DailyBar、数据同步任务和股票数据查询的业务流程。
- 按 Domain、Application、Interface、Adapter 和配置边界组织实现。

## 边界与验证

- Domain 保持业务纯度；Application 编排用例；Interface 处理 HTTP；Adapter 承载 GORM Repository 和外部数据源。
- 可依赖共享 internal/infrastructure，但共享 Infrastructure 不得依赖本目录。
- 最小验证：执行 go test -mod=readonly ./internal/data/... -count=1。
