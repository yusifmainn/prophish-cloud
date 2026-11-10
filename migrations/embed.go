// Package migrations embeds the SQL schema migrations so the application can
// apply them itself at startup — the single source of truth for the DB schema.
// (Postgres initdb ignores non-SQL files, so this .go file is harmless even if
// the directory is ever mounted into /docker-entrypoint-initdb.d.)
package migrations

import "embed"

// FS holds every *.sql migration in this directory, applied in filename order.
//
//go:embed *.sql
var FS embed.FS
