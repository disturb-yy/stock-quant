# INDEX.md

## Package

`internal/research`

## Role

RES-001 Research 项目创建、最近研究列表和基础元数据读取的 Application、Interface 边界。

## Current Contents

- `domain/`：Research 项目实体、可空描述和名称/描述校验。
- `service.go`：创建、按最近更新时间分页列表和详情读取用例编排。
- `handler.go`：`/api/v1/research` 创建、列表和详情 HTTP 适配。
- `infrastructure/`：`t_research_workspace` 的 MySQL migration、持久化和幂等 Demo Seed。
