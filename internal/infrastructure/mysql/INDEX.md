# MySQL 基础设施导航

| 文件 | 职责 |
|---|---|
| `doc.go` | 声明 MySQL 适配器包 |
| `migrate.go` | 加载版本化迁移、串行执行并跟踪 checksum/dirty 状态 |
| `migrate_test.go` | MySQL 8.4 迁移集成测试 |
| `run_repositories.go` | 同步任务、选股运行/结果及回测运行仓储 |
| `run_repositories_test.go` | MySQL 8.4 幂等键、状态终态、事务回滚和排序集成测试 |
| `repositories.go` | 股票、交易日历、日线和复权因子的事务仓储；行情 DECIMAL 使用文本绑定和精确读取 |
| `repositories_test.go` | MySQL 8.4 仓储集成测试，覆盖修订、回滚、历史查询、日期索引和高精度 DECIMAL 往返 |
