# AGENTS.md

## Scope

本目录是 Research 项目的 Application 与 HTTP Interface 边界，负责项目创建、最近列表和基础元数据读取。

## Rule

`domain` 只放 Research 项目身份与输入校验；MySQL、迁移和 Demo Seed 只能位于 `infrastructure`。Handler 只解析已发布的 HTTP 字段，不接收客户端提供的 ID 或时间。
