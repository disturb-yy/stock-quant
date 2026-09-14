# AGENTS.md

## Scope

本 package 是数据库 migration 的技术边界。

## Rule

只提供 migration 的命名、注册和执行抽象；具体数据库 Driver、连接和业务表 migration 由后续基础设施需求接入。

禁止：

- 创建本 US 范围外的业务表。
- 依赖 HTTP、Gin 或业务领域 package。
- 将 migration 逻辑放入 Handler。
