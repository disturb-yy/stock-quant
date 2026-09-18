# AGENTS.md

## Scope

本目录是股票池的 MySQL Infrastructure 边界。

## Rule

SQL 必须参数化；migration、查询和 Demo Seed 只能操作 `t_stock_pool` 及 migration 元数据。不得在这里实现 HTTP 或领域校验规则。
