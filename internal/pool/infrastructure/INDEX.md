# INDEX.md

## Package

`internal/pool/infrastructure`

## Role

实现 POL-001 的 `t_stock_pool` MySQL migration、持久化和 Demo Seed。

## Current Contents

- `mysql.go`：`0009_stock_pool` 幂等 migration、手工池创建、名称搜索、稳定分页、详情读取和可重复 Demo Seed。
- `mysql_test.go`：搜索转义与存储边界的纯单元测试。
- `mysql_integration_test.go`：显式 MySQL DSN 下的 migration、Seed、创建、搜索、分页和详情验证。
