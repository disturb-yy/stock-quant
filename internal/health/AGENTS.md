# AGENTS.md

## Scope

本 package 是跨领域健康 HTTP 接口适配层。

## Responsibilities

允许：

- 注册存活心跳和服务探针路由。
- 输出健康接口的 HTTP 状态码与响应 DTO。

禁止：

- 编写业务规则。
- 直接访问数据库、缓存或外部 API。
- 在 `cmd/server` 中复制健康接口处理逻辑。
