# INDEX.md

## Package

`internal/pool/infrastructure`

## Role

实现 POL-001/POL-002 的 `t_stock_pool`、`t_stock_pool_member` MySQL migration、持久化和 Demo Seed。

## Current Contents

- `mysql.go`：`0009_stock_pool`/`0010_stock_pool_member` 幂等 migration、手工池创建、名称搜索、成员增删读、稳定分页、准确成员数和可重复 Demo Seed。
- `mysql_summary.go`：在只读事务内读取来源引用、行业分布、PE/ROE 摘要及空值/依赖语义；不写入 Screener 来源。
- `mysql_test.go`：搜索转义与存储边界的纯单元测试。
- `mysql_integration_test.go`：显式 MySQL DSN 下的 migration、Seed、创建、搜索、分页、详情、summary 画像和成员唯一性验证。
