# migrations 局部规约

本目录保存 A 股数据 bounded context 的版本化 schema 资源。遵循根 AGENTS.md。

## 职责

- 保存可按版本执行的 up 与 down SQL。
- 通过 embed.FS 向共享 migration runner 提供资源。

## 边界与验证

- SQL 只描述当前数据 schema，不在此目录实现 Repository、连接管理或业务逻辑。
- 每个 up 变更必须有对应可逆的 down 变更，并保持版本文件命名一致。
- 最小验证：执行 go test -mod=readonly ./migrations -count=1。
