//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/contracts"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// A domain service's decision goes out through the outbox, like an
// aggregate's events (ADR-013).
func TestEventPublisher_WritesToTheOutbox(t *testing.T) {
	pool := pgtest.New(t)
	show := domain.NewShowID(uuid.New())

	err := postgres.NewEventPublisher(pool).Publish(context.Background(), domain.InventorySoldOut{ShowID: show, At: now})

	if err != nil {
		t.Fatal(err)
	}
	var typ string
	if err := pool.QueryRow(context.Background(),
		`SELECT event_type FROM ticketing.outbox WHERE msg_key = $1`, show.String()).Scan(&typ); err != nil || typ != contracts.TypeInventorySoldOutV1 {
		t.Errorf("outbox row = %q, %v; want %s", typ, err, contracts.TypeInventorySoldOutV1)
	}
}
