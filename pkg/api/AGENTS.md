# AGENTS.md

## Scope

本 package 是跨领域 API 契约边界。

## Responsibilities

允许：

- 定义可复用的成功、错误和分页 DTO。
- 定义稳定错误码与纯 Go 的参数解析规则。
- 构造服务实际输出的 OpenAPI 文档。

禁止：

- 依赖 Gin、数据库、配置或具体业务领域。
- 编写 HTTP 路由、业务规则或基础设施实现。
