# AGENTS.md

## Scope

本目录是量化选股的 MySQL 快照查询适配边界。

## Rule

仅复用既有 FND-003、Markets、STK-002、STK-003、STK-004 表；SQL 必须静态参数化，禁止把用户字段名或表达式拼入 SQL。
