# INDEX.md

## Package

`internal/research/infrastructure`

## Role

实现 RES-001 的 `t_research_workspace` MySQL migration、项目读写和可重复 Demo Seed。

## Current Contents

- `mysql.go`：`0011_research_workspace` 幂等 migration、创建/最近列表/详情查询和两个项目的 Seed。
- `mysql_integration_test.go`：显式 MySQL DSN 下验证 migration、Seed 幂等、创建、详情和更新时间排序。
