# internal/data/config 局部规约

本目录保存 A 股数据 bounded context 的运行配置。遵循父级 internal/data/AGENTS.md 与根 AGENTS.md。

## 职责

- 读取和校验 Provider、Tushare Endpoint、Token、重试和同步 Worker 配置。
- 将配置转换为 data Application 和 Adapter 使用的结构。

## 边界与验证

- 只处理 A 股数据领域配置；共享数据库 DSN 由 internal/config 处理。
- 配置在 Composition Root 读取并注入，不在 Domain 或 Adapter 中自行读取环境变量。
- 最小验证：执行 go test -mod=readonly ./internal/data/config -count=1。
