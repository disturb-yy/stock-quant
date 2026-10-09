# MySQL 基础设施导航

| 文件 | 职责 |
|---|---|
| `doc.go` | 声明 MySQL 适配器包 |
| `migrate.go` | 加载版本化迁移、串行执行并跟踪 checksum/dirty 状态 |
| `migrate_test.go` | MySQL 8.4 迁移集成测试 |
| `repositories.go` | 股票、交易日历、日线和复权因子的事务仓储与有序查询 |
| `repositories_test.go` | MySQL 8.4 仓储集成测试，覆盖修订、回滚、历史查询和日期索引 |
