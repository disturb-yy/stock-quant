# migrations 导航

## 目录职责

保存 A 股数据 schema 的版本化迁移资源。

## 当前文件

- 000001_create_data_sync_tables.up.sql：创建数据同步相关表。
- 000001_create_data_sync_tables.down.sql：回滚数据同步相关表。
- embed.go：将迁移 SQL 嵌入为 embed.FS。
- runner_test.go：验证迁移资源与 runner 的协作行为。
