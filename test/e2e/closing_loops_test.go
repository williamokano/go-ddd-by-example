//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// showStatus reads a show's status from Show's API.
func (c *client) showStatus(id string) string {
	c.t.Helper()
	var s showJSON
	mustStatus(c.t, c.do(http.MethodGet, "/shows/"+id, nil), http.StatusOK).decode(c.t, &s)
	return s.Status
}

// S4 — Sold out: selling the last unit makes Show mark the show sold out
// (TKT-10 → ticketing.inventory_sold_out.v1 → SHW-8).
func TestS4_SoldOut(t *testing.T) {
	c := newClient(t)
	show := c.publishedShow()
	order := c.buy(show, "ORCH/A/1", "ORCH/A/2", "FLOOR/GA/0001", "FLOOR/GA/0002", "FLOOR/GA/0003")

	eventually(t, 20*time.Second, "the order is fulfilled", func() bool {
		return c.order(order).Status == "fulfilled"
	})
	eventually(t, 20*time.Second, "the show is sold out", func() bool {
		return c.showStatus(show) == "sold_out"
	})
}

// A cancelled show closes its inventory, voids its tickets and refunds its
// paid orders (TKT-11), each step its own transaction driven by events.
func TestShowCancellation_VoidsTicketsAndRefunds(t *testing.T) {
	c := newClient(t)
	promoter := uuid.NewString()
	show := c.publishedShowBy(promoter)
	order := c.buy(show, "ORCH/A/1")
	eventually(t, 20*time.Second, "the order is fulfilled", func() bool {
		return c.order(order).Status == "fulfilled"
	})

	mustStatus(t, c.do(http.MethodPost, "/shows/"+show+"/cancellation", map[string]any{
		"promoterId": promoter, "reason": "the band split up",
	}), http.StatusNoContent)

	var got orderJSON
	eventually(t, 20*time.Second, "the order is refunded and its ticket voided", func() bool {
		got = c.order(order)
		return got.Status == "refunded" && len(got.Tickets) == 1 && got.Tickets[0].Status == "voided"
	})
	r := c.do(http.MethodPost, "/shows/"+show+"/holds", map[string]any{"customerId": uuid.NewString(), "seats": []string{"ORCH/A/2"}})
	if r.Status != http.StatusConflict {
		t.Errorf("hold on a cancelled show: status %d, want 409", r.Status)
	}
}
