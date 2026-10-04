//go:build integration

package db_test

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

// asRole connects as a context's role (dev credentials: password = role).
func asRole(t *testing.T, role string) *pgx.Conn {
	t.Helper()
	u, err := url.Parse(pgtest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword(role, role)
	conn, err := pgx.Connect(context.Background(), u.String())
	if err != nil {
		t.Fatalf("connect as %s: %v", role, err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

// 9.6: the dependency rule, enforced a second time by the database. Each
// context's role reads and writes its own schema and nothing else, so a
// cross-schema join fails even if a query slips past the architecture test.
func TestEachContextRoleSeesOnlyItsSchema(t *testing.T) {
	tables := map[string]string{
		"venue": "venue.venues", "show": "show.shows", "ticketing": "ticketing.orders", "notifications": "notifications.inbox",
	}
	for ctx := range tables {
		t.Run(ctx, func(t *testing.T) {
			conn := asRole(t, "stagehand_"+ctx)
			for other, table := range tables {
				_, err := conn.Exec(context.Background(), "SELECT 1 FROM "+table+" LIMIT 1")
				switch {
				case other == ctx && err != nil:
					t.Errorf("own schema %s: %v", table, err)
				case other != ctx && (err == nil || !strings.Contains(err.Error(), "permission denied")):
					t.Errorf("%s read %s: err = %v, want permission denied", ctx, table, err)
				}
			}
		})
	}
}

func TestAContextRoleCanWriteItsOutbox(t *testing.T) {
	conn := asRole(t, "stagehand_show")
	_, err := conn.Exec(context.Background(),
		`INSERT INTO show.outbox (event_id, topic, msg_key, event_type, payload, occurred_at)
		 VALUES (gen_random_uuid(), 't', 'k', 'x', '{}', now())`)
	if err != nil {
		t.Errorf("insert into its outbox (needs the sequence too): %v", err)
	}
}
