# internal/data/domain 局部规约

本目录表达 A 股数据 bounded context 的业务概念和规则。遵循父级 internal/data/AGENTS.md 与根 AGENTS.md。

## 职责

- 定义 Instrument、DailyBar、SyncTask、DateRange 和查询模型。
- 定义 Repository、StockDataSource 等抽象接口和领域错误。
- 执行领域校验、状态转换和不变量约束。

## 边界与验证

- 不依赖 Gin、GORM、数据库驱动、外部 Provider 或运行配置。
- Domain 测试应能在不启动 HTTP Server、数据库或外部服务的情况下运行。
- 最小验证：执行 go test -mod=readonly ./internal/data/domain -count=1。
