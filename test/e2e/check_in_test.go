//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TKT-13 end to end: a ticket bought for tonight's show is let in once.
func TestCheckIn_OnceOnTheShowsDay(t *testing.T) {
	c := newClient(t)
	promoter := uuid.NewString()
	show := c.draftShow(c.activeVenue(), promoter, schedule(2*time.Hour))
	c.priceAndPublish(show, promoter)
	order := c.buy(show, "ORCH/A/1")

	var got orderJSON
	eventually(t, 20*time.Second, "the ticket is issued", func() bool {
		got = c.order(order)
		return got.Status == "fulfilled" && len(got.Tickets) == 1
	})
	path := "/tickets/" + got.Tickets[0].Code + "/check-in"

	mustStatus(t, c.do(http.MethodPost, path, map[string]any{"gateId": "north-1"}), http.StatusNoContent)
	mustStatus(t, c.do(http.MethodPost, path, map[string]any{"gateId": "south-2"}), http.StatusConflict)
	if s := c.order(order).Tickets[0].Status; s != "checked_in" {
		t.Errorf("ticket status = %q, want checked_in", s)
	}
}
