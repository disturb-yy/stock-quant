# internal/infrastructure 局部规约

本目录保存跨 bounded context 共享的技术 Infrastructure。遵循父级 internal/AGENTS.md 与根 AGENTS.md。

## 职责

- 提供真实存在多个使用方的技术能力。
- 当前包括 GORM MySQL 连接生命周期和通用 migration runner。

## 边界与验证

- 只能依赖标准库或第三方技术库，不依赖任何具体 bounded context 的模型、表或业务接口。
- 领域特有 SQL、Provider 响应转换和 Repository 实现放回对应 internal/<bounded-context>/adapter。
- 最小验证：执行 go test -mod=readonly ./internal/infrastructure/... -count=1。
