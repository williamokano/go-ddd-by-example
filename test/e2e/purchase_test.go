//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

type holdJSON struct {
	HoldID    string    `json:"holdId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type orderJSON struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Total  struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	} `json:"total"`
	Tickets []struct {
		Seat string `json:"seat"`
		Code string `json:"code"`
	} `json:"tickets"`
}

// publishedShow activates a venue (ORCH: A1–A2 seated; FLOOR: 3 GA places)
// and publishes a show there a month from now.
func (c *client) publishedShow() string {
	c.t.Helper()
	promoter := uuid.NewString()
	show := c.draftShow(c.activeVenue(), promoter, schedule(30*24*time.Hour))
	c.priceAndPublish(show, promoter)
	return show
}

// hold holds seats, waiting until Ticketing has opened the inventory
// (show.published.v1 travels through Kafka).
func (c *client) hold(showID, customerID string, seats ...string) holdJSON {
	c.t.Helper()
	var h holdJSON
	eventually(c.t, 20*time.Second, "the inventory opens", func() bool {
		r := c.do(http.MethodPost, "/shows/"+showID+"/holds", map[string]any{"customerId": customerID, "seats": seats})
		if r.Status == http.StatusCreated {
			r.decode(c.t, &h)
			return true
		}
		return false
	})
	return h
}

// S1 — Happy purchase.
func TestS1_HappyPurchase(t *testing.T) {
	c := newClient(t)
	show := c.publishedShow()
	customer := uuid.NewString()
	h := c.hold(show, customer, "ORCH/A/1", "ORCH/A/2")

	var placed struct{ ID string }
	mustStatus(t, c.do(http.MethodPost, "/orders", map[string]any{
		"holdId": h.HoldID, "customerId": customer, "contactEmail": "ana@example.com",
	}), http.StatusCreated).decode(t, &placed)

	var order orderJSON
	eventually(t, 20*time.Second, "the order is fulfilled with 2 tickets", func() bool {
		mustStatus(t, c.do(http.MethodGet, "/orders/"+placed.ID, nil), http.StatusOK).decode(t, &order)
		return order.Status == "fulfilled" && len(order.Tickets) == 2
	})
	if order.Total.Amount != 9000 || order.Total.Currency != "EUR" {
		t.Errorf("total = %+v, want EUR 90.00", order.Total)
	}
	if order.Tickets[0].Code == order.Tickets[1].Code {
		t.Errorf("ticket codes are not unique: %+v (TKT-9)", order.Tickets)
	}
	var seats []struct {
		Ref   string `json:"ref"`
		State string `json:"state"`
	}
	mustStatus(t, c.do(http.MethodGet, "/shows/"+show+"/seats", nil), http.StatusOK).decode(t, &seats)
	sold := 0
	for _, s := range seats {
		if s.State == "sold" {
			sold++
		}
	}
	if sold != 2 {
		t.Errorf("%d seats sold, want 2", sold)
	}
}
