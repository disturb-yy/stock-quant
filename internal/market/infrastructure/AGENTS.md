# AGENTS.md

## Scope

本目录是行情领域的 MySQL 查询适配边界。

## Rule

只实现 `internal/market` 定义的数据访问接口；SQL、数据库字段和扫描逻辑留在本目录，不向 Application 或 Handler 泄露数据库细节。
