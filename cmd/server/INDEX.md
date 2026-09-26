# cmd/server 导航

## 目录职责

HTTP 服务启动入口与依赖组合。

## 当前文件

- AGENTS.md：本目录的启动、装配和边界规约。
- main.go：加载配置，创建 GORM MySQL 连接，组装 A 股数据 Adapter 与 Application Service，并处理服务生命周期。
- main_test.go：验证服务启动入口的配置错误和生命周期行为。
- router.go：创建 Gin Router，安装全局中间件，创建 /api/v1 路由组并注册健康检查、数据同步与查询路由。
- router_test.go：验证全局路由和中间件装配。
- INDEX.md：本目录导航。
