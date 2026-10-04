// Command stagehand is the whole platform in one binary (ADR-001).
//
//	stagehand serve                          run the HTTP API
//	stagehand healthcheck                    exit 0 if the local API answers (for Docker)
//	stagehand migrate up|down|reset|status   apply the embedded migrations to $DATABASE_URL
package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/williamokano/go-ddd-by-example/db"
	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Args[1:])
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "stagehand:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	switch {
	case len(args) == 1 && args[0] == "serve":
		return serve(ctx)
	case len(args) == 1 && args[0] == "healthcheck":
		return healthcheck(ctx)
	case len(args) == 2 && args[0] == "migrate":
		migrations, err := fs.Sub(db.Migrations, "migrations")
		if err != nil {
			return fmt.Errorf("migrations: %w", err)
		}
		if err := postgres.Migrate(ctx, os.Getenv("DATABASE_URL"), migrations, args[1]); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("usage: stagehand serve | healthcheck | migrate up|down|reset|status")
	}
}

// healthcheck asks the local API for /healthz; distroless images have no curl.
func healthcheck(ctx context.Context) error {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost"+addr+"/healthz", nil)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("healthcheck: status %d", resp.StatusCode)
	}
	return nil
}
