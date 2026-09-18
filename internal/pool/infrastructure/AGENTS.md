# AGENTS.md

## Scope

本目录是股票池及其成员的 MySQL Infrastructure 边界。

## Rule

SQL 必须参数化；migration、查询和 Demo Seed 只能操作 `t_stock_pool`、`t_stock_pool_member` 及其已存在的 `instruments.code` 身份引用和 migration 元数据。不得在这里实现 HTTP 或领域校验规则。
