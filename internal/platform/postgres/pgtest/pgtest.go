//go:build integration

// Package pgtest starts one real Postgres per test package (testcontainers),
// applies the embedded migrations, and hands tests a connection pool.
//
//	func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }
//	func TestX(t *testing.T)    { pool := pgtest.New(t) ... }
//
// One container per package, not per test: tests isolate themselves with
// fresh random IDs instead.
package pgtest

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/williamokano/go-ddd-by-example/db"
	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
)

var databaseURL string

// Main starts the container, migrates it up, runs the tests and stops it.
func Main(m *testing.M) int {
	ctx := context.Background()
	container, err := tcpostgres.Run(ctx, "postgres:18",
		tcpostgres.WithDatabase("stagehand"),
		tcpostgres.WithUsername("stagehand"),
		tcpostgres.WithPassword("stagehand"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Printf("pgtest: start postgres: %v", err)
		return 1
	}
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			log.Printf("pgtest: stop postgres: %v", err)
		}
	}()

	databaseURL, err = container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Printf("pgtest: connection string: %v", err)
		return 1
	}
	if err := migrateUp(ctx); err != nil {
		log.Printf("pgtest: %v", err)
		return 1
	}
	return m.Run()
}

func migrateUp(ctx context.Context) error {
	migrations, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	if err := postgres.Migrate(ctx, databaseURL, migrations, "up"); err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// URL returns the database URL of the package's container.
func URL(t *testing.T) string {
	t.Helper()
	if databaseURL == "" {
		t.Fatal("pgtest: no database; call pgtest.Main from TestMain")
	}
	return databaseURL
}

// New returns a pool on the package's (migrated) database, closed when the
// test ends.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), URL(t))
	if err != nil {
		t.Fatalf("pgtest: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
