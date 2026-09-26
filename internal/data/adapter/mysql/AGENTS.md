# internal/data/adapter/mysql 局部规约

本目录实现 A 股数据 bounded context 的 MySQL Repository。遵循父级 internal/data/adapter/AGENTS.md 与根 AGENTS.md。

## 职责

- 使用 GORM 查询数据同步任务、Instrument 和 DailyBar。
- 将持久化 Record 与 Domain 模型互相映射。
- 维护事务、锁、分页、upsert 和领域错误映射语义。

## 边界与验证

- 可以依赖 data/domain 和共享 GORM 数据库能力。
- 表结构必须与 migrations 中的 schema 一致；不在此目录修改迁移 SQL。
- 最小验证：执行 go test -mod=readonly ./internal/data/adapter/mysql -count=1。
