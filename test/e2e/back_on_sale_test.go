//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// 9.5: a sold-out show gets a return (TKT-14); its seats go back on sale,
// the tickets are voided, and Show publishes the show again (SHW-10).
func TestReturn_PutsASoldOutShowBackOnSale(t *testing.T) {
	c := newClient(t)
	show := c.publishedShow()
	customer := uuid.NewString()
	orch := c.checkout(c.hold(show, customer, "ORCH/A/1", "ORCH/A/2"), customer, nil)
	c.buy(show, "FLOOR/GA/0001", "FLOOR/GA/0002", "FLOOR/GA/0003")
	eventually(t, 20*time.Second, "the show is sold out", func() bool {
		return c.showStatus(show) == "sold_out"
	})

	mustStatus(t, c.do(http.MethodPost, "/orders/"+orch+"/return", map[string]any{"customerId": customer}), http.StatusAccepted)

	eventually(t, 20*time.Second, "the show is published again", func() bool {
		return c.showStatus(show) == "published"
	})
	if s := c.seatStates(show)["ORCH/A/1"]; s != "available" {
		t.Errorf("ORCH/A/1 is %s, want available", s)
	}
	got := c.order(orch)
	if got.Status != "refunded" || len(got.Tickets) != 2 || got.Tickets[0].Status != "voided" {
		t.Errorf("order = %s with %+v; want refunded, tickets voided", got.Status, got.Tickets)
	}
}
