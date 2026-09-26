# internal/data/adapter/mysql 导航

## 目录职责

使用 GORM 实现 A 股数据持久化。

## 当前文件

- records.go：定义 GORM 持久化 Record 及 Domain 映射。
- repository.go：实现数据同步任务和股票数据查询 Repository。
- repository_test.go：验证查询、事务、锁、upsert 和错误映射。
