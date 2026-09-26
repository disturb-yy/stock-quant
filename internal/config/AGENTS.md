# internal/config 局部规约

本目录保存应用运行配置的读取与校验。遵循父级 internal/AGENTS.md 与根 AGENTS.md。

## 职责

- 从 Composition Root 读取共享运行配置。
- 在启动阶段校验配置，并向调用方返回可诊断但不泄露敏感值的错误。

## 边界与验证

- 只维护跨领域运行配置；A 股数据源配置放在 internal/data/config。
- 业务 Domain、Application 和 Adapter 不得自行读取环境变量。
- 最小验证：执行 go test -mod=readonly ./internal/config -count=1。
