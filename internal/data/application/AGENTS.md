# internal/data/application 局部规约

本目录负责 A 股数据用例编排。遵循父级 internal/data/AGENTS.md 与根 AGENTS.md。

## 职责

- 编排创建、查询、重试和执行数据同步任务。
- 组织股票数据查询和后台 Worker 的执行流程。
- 调用 Domain 接口，不暴露持久化 Record 或 Provider 响应。

## 边界与验证

- 不解析 HTTP 参数、不直接使用 Gin、GORM、SQL 或外部 Provider SDK。
- 事务边界和重试语义在 Application 与 Domain 之间保持可测试。
- 最小验证：执行 go test -mod=readonly ./internal/data/application -count=1。
