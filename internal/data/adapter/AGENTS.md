# internal/data/adapter 局部规约

本目录保存 A 股数据 bounded context 特有的技术适配器。遵循父级 internal/data/AGENTS.md 与根 AGENTS.md。

## 职责

- 实现 Domain 声明的 Repository 和 Provider 接口。
- 负责 GORM Record 映射、SQL 查询语义和外部 Provider 响应转换。

## 边界与验证

- 可以依赖本 bounded context 的 Domain 和共享 Infrastructure。
- 不承载 Application 编排、HTTP 参数解析或跨领域流程。
- 最小验证：执行 go test -mod=readonly ./internal/data/adapter/... -count=1。
