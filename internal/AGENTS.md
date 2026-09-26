# internal 局部规约

本目录保存服务内部实现，外部项目不得导入。遵循根 AGENTS.md；各子目录的局部文档只补充本层边界。

## 职责

- 按 bounded context 组织业务实现。
- 提供共享技术 Infrastructure、运行配置、健康检查和领域内部层次。

## 边界与验证

- Domain、Application、Interface、Adapter 和共享 Infrastructure 遵循根规约中的依赖方向。
- 跨 bounded context 通过公开接口或 Application Service 协作，不直接依赖对方具体 Adapter。
- 最小验证：执行 go test -mod=readonly ./internal/... -count=1。
