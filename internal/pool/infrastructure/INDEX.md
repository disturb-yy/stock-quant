# INDEX.md

## Package

`internal/pool/infrastructure`

## Role

实现 POL-001/POL-002 的 `t_stock_pool`、`t_stock_pool_member` MySQL migration、持久化和 Demo Seed。

## Current Contents

- `mysql.go`：`0009_stock_pool`/`0010_stock_pool_member` 幂等 migration、手工池创建、名称搜索、成员增删读、稳定分页、准确成员数和可重复 Demo Seed。
- `mysql_test.go`：搜索转义与存储边界的纯单元测试。
- `mysql_integration_test.go`：显式 MySQL DSN 下的 migration、Seed、创建、搜索、分页、详情和成员唯一性验证。
