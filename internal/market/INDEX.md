# INDEX.md

## Package

`internal/market`

## Role

日线行情领域和本地 Provider 模式选择。

## Current Contents

- `domain/daily_bar.go`：Daily Bar 实体和最小领域校验。
- `provider.go`：demo/real/fallback 模式及 Provider 名称解析；当前 real 仅为未来实现保留。
