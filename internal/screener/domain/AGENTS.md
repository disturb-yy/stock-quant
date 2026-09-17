# AGENTS.md

## Scope

本目录是量化选股的纯业务规则边界。

## Rule

不得依赖 Gin、MySQL、Driver、配置或具体 Provider；输入通过领域模型传入，缺失值保持 nullable 语义。
