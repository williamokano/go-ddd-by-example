// Command stagehand is the whole platform in one binary (ADR-001).
//
//	stagehand migrate up|down|reset|status   apply the embedded migrations to $DATABASE_URL
package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"

	"github.com/williamokano/go-ddd-by-example/db"
	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "stagehand:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 2 && args[0] == "migrate" {
		migrations, err := fs.Sub(db.Migrations, "migrations")
		if err != nil {
			return fmt.Errorf("migrations: %w", err)
		}
		if err := postgres.Migrate(ctx, os.Getenv("DATABASE_URL"), migrations, args[1]); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
		return nil
	}
	return fmt.Errorf("usage: stagehand migrate up|down|reset|status")
}
