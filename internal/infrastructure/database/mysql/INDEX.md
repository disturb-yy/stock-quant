# internal/infrastructure/database/mysql 导航

## 目录职责

使用 GORM MySQL Dialector 提供共享 MySQL 连接。

## 当前文件

- open.go：创建、Ping 和配置 GORM MySQL 连接。
- open_test.go：验证 DSN 校验和连接初始化错误语义。
