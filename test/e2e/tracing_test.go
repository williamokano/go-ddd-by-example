//go:build e2e

package e2e_test

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// outboxRow is one row of ticketing.outbox, as the tracing test reads it.
type outboxRow struct {
	EventID, Type, Causation string
}

// ticketingOutbox reads the purchase's rows straight from Postgres: the IDs
// are an operational concern with no API of their own.
func ticketingOutbox(t *testing.T, correlation string) []outboxRow {
	t.Helper()
	url := os.Getenv("STAGEHAND_DATABASE_URL")
	if url == "" {
		url = "postgres://stagehand:stagehand@localhost:5432/stagehand?sslmode=disable"
	}
	conn, err := pgx.Connect(t.Context(), url)
	if err != nil {
		t.Fatalf("connect to Postgres: %v", err)
	}
	defer func() { _ = conn.Close(context.Background()) }()
	rows, err := conn.Query(t.Context(),
		`SELECT event_id::text, event_type, causation_id FROM ticketing.outbox WHERE correlation_id = $1 ORDER BY id`, correlation)
	if err != nil {
		t.Fatal(err)
	}
	out, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (outboxRow, error) {
		var o outboxRow
		return o, r.Scan(&o.EventID, &o.Type, &o.Causation)
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// S1, traced: one correlation ID from the customer's first request to the
// last event of the saga, and every event caused by the one before (8.3).
func TestS1_OnePurchaseIsOneFlow(t *testing.T) {
	c := newClient(t)
	show := c.publishedShow()
	customer := uuid.NewString()
	flow := map[string]string{"X-Correlation-ID": "purchase-" + uuid.NewString()}
	h := c.hold(show, customer, "ORCH/A/1")
	order := c.checkout(h, customer, flow)

	want := []string{"ticketing.order_paid", "ticketing.seats_sold", "ticketing.tickets_issued.v1"}
	var rows []outboxRow
	eventually(t, 20*time.Second, "the saga's events share the purchase's correlation ID", func() bool {
		rows = ticketingOutbox(t, flow["X-Correlation-ID"])
		var types []string
		for _, r := range rows {
			types = append(types, r.Type)
		}
		return slices.Equal(types, want)
	})
	if rows[0].Causation != "" {
		t.Errorf("order_paid comes from HTTP, yet has causation %q", rows[0].Causation)
	}
	for i := 1; i < len(rows); i++ {
		if rows[i].Causation != rows[i-1].EventID {
			t.Errorf("%s caused by %q, want the previous event %s", rows[i].Type, rows[i].Causation, rows[i-1].EventID)
		}
	}
	if got := c.order(order).Status; got != "fulfilled" {
		t.Errorf("order is %s, want fulfilled", got)
	}
}
