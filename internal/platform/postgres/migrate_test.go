//go:build integration

package postgres_test

import (
	"context"
	"io/fs"
	"os"
	"testing"

	"github.com/williamokano/go-ddd-by-example/db"
	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

// Catches broken Down sections early: every migration must be reversible.
func TestMigrate_UpResetUp(t *testing.T) {
	ctx := context.Background()
	migrations, err := fs.Sub(db.Migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	url := pgtest.URL(t)

	for _, command := range []string{"up", "reset", "up"} {
		if err := postgres.Migrate(ctx, url, migrations, command); err != nil {
			t.Fatalf("migrate %s: %v", command, err)
		}
	}
}
