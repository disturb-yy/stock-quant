# internal/data/adapter/source 局部规约

本目录实现 A 股数据源 Adapter。遵循父级 internal/data/adapter/AGENTS.md 与根 AGENTS.md。

## 职责

- 根据数据源配置创建 Tushare 或 Mock Adapter。
- 将外部数据源响应转换为 data/domain 模型，并标记 SourceProvenance。

## 边界与验证

- 可以使用标准 HTTP Client 和 data/domain 类型。
- 不写数据库、不创建 HTTP 服务路由、不绕过 Application Service。
- 最小验证：执行 go test -mod=readonly ./internal/data/adapter/source -count=1。
