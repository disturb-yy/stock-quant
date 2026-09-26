# cmd/migrate 导航

## 目录职责

执行版本化数据库 schema 迁移。

## 当前文件

- main.go：读取配置、创建 GORM MySQL 连接并执行 migration runner。
- main_test.go：验证迁移命令入口的配置错误和生命周期行为。
