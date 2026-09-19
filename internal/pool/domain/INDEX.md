# INDEX.md

## Package

`internal/pool/domain`

## Role

定义 POL-001 股票池的手工来源、不变量与输入校验。

## Current Contents

- `model.go`：股票池实体、成员身份、手工来源、名称/描述/symbol 校验规则和业务错误。
- `summary.go`：来源追溯、行业分布、PE/ROE 摘要及空值/可用性语义。
