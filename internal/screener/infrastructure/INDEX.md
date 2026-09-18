# INDEX.md

## Package

`internal/screener/infrastructure`

## Role

读取统一执行快照、Universe 身份、行业关系和已存在行情/估值/财务字段。

## Current Contents

- `mysql.go`：单次批量 MySQL 查询、来源元数据和字段 as_of 组装。
- `mysql_saved.go`：`0008_screener_saved` 幂等 migration，以及方案当前身份/不可变版本的事务 CRUD。
- `mysql_saved_test.go`：规范化 spec 的 JSON 序列化/反序列化和未知字段保护。
