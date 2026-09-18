# AGENTS.md

## Scope

本目录是股票池的 Application 与 HTTP Interface 边界，负责创建、列表、概览读取和成员管理。

## Rule

`domain` 只放股票池身份、来源、成员身份和输入校验规则；MySQL 与迁移只能位于 `infrastructure`。Handler 只解析已发布的 HTTP 字段，不能接收客户端提供的来源、成员数、ID 或时间。
