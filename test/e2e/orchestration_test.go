//go:build e2e

package e2e_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// 9.4: with SAGA_STYLE=orchestration (the default), every checkout has a
// process that says where it is. Run the suite with SAGA_STYLE=choreography
// to compare: S1–S6 pass either way.
func TestS1_OrchestratedCheckoutIsCompleted(t *testing.T) {
	if os.Getenv("SAGA_STYLE") == "choreography" {
		t.Skip("choreography keeps no process")
	}
	c := newClient(t)
	show := c.publishedShow()
	customer := uuid.NewString()
	order := c.checkout(c.hold(show, customer, "ORCH/A/1"), customer, nil)

	conn, err := pgx.Connect(t.Context(), databaseURL())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(context.Background()) }()
	var state string
	eventually(t, 20*time.Second, "the checkout process completes", func() bool {
		err := conn.QueryRow(t.Context(), `SELECT state FROM ticketing.checkout_processes WHERE order_id = $1`, order).Scan(&state)
		return err == nil && state == "completed"
	})
	if got := c.order(order).Status; got != "fulfilled" {
		t.Errorf("order is %s, want fulfilled", got)
	}
}
