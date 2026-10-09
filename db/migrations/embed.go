// Package migrations embeds versioned SQL migration files.
package migrations

import "embed"

// FS contains the versioned up/down SQL files used by the MySQL migration runner.
//
//go:embed *.up.sql *.down.sql
var FS embed.FS
