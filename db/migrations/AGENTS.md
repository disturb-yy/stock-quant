# 数据库迁移规约

- 迁移按递增版本命名为 `NNNN.up.sql` 和 `NNNN.down.sql`，提交后保持内容不可变。
- 每个方向的语句必须允许在失败后安全重跑；MySQL DDL 不是事务原子操作。
- `down` 只能删除本迁移创建的对象，并保留迁移版本表。
- 新迁移须同步 `docs/05-database.md` 和 MySQL 集成测试。
