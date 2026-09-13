# AGENTS.md

## Scope

本 package 是服务 Composition Root。

## Responsibilities

允许：

- 加载配置。
- 初始化日志。
- 创建数据库、Redis、外部 API Client。
- 创建 Repository / Provider 实现。
- 注入 Application Service。
- 注册 Handler。
- 启动和关闭服务。

禁止：

- 编写业务规则。
- 写 Repository SQL。
- 实现股票、行情、分析逻辑。

`main.go` 应保持薄，只负责组装。
