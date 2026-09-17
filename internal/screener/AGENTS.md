# AGENTS.md

## Scope

本目录是量化选股执行领域，负责结构化规格校验、条件组合、排序和结果语义。

## Rule

`domain` 只放选股规则、字段 registry 和执行结果模型；SQL、数据库字段和扫描逻辑留在 `infrastructure`。Handler 只做 JSON/HTTP 适配，Application 负责快照读取与领域执行编排。
