package migrations

import "embed"

// FS 暴露 A 股数据 domain 的 schema 资源，由共享 migration runner 执行。
//
//go:embed *.sql
var FS embed.FS
