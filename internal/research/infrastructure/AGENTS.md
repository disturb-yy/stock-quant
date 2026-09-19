# AGENTS.md

## Scope

本目录是 Research 项目的 MySQL Infrastructure 边界。

## Rule

SQL 必须参数化；migration、查询和 Demo Seed 只能操作 `t_research_workspace` 及 schema migration 元数据，不实现 HTTP 或领域校验规则。
