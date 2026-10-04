// Package db embeds the SQL migrations, so the migrate subcommand, the
// integration tests and CI all apply exactly the same files.
package db

import "embed"

// Migrations holds db/migrations/*.sql (goose format).
//
//go:embed migrations/*.sql
var Migrations embed.FS
