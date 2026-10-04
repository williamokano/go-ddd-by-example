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
		Seat   string `json:"seat"`
		Code   string `json:"code"`
		Status string `json:"status"`
	} `json:"tickets"`
}

// publishedShow activates a venue (ORCH: A1–A2 seated; FLOOR: 3 GA places)
// and publishes a show there a month from now.
func (c *client) publishedShow() string {
	c.t.Helper()
	return c.publishedShowBy(uuid.NewString())
}

// publishedShowBy is publishedShow for a promoter the test needs to know.
func (c *client) publishedShowBy(promoter string) string {
	c.t.Helper()
	return c.publishedShowAt(c.activeVenue(), promoter)
}

// publishedShowAt publishes a show a month from now at the given venue.
func (c *client) publishedShowAt(venueID, promoter string) string {
	c.t.Helper()
	show := c.draftShow(venueID, promoter, schedule(30*24*time.Hour))
	c.priceAndPublish(show, promoter)
	return show
}

// buy holds the seats and places an order for them, and returns the order id.
func (c *client) buy(showID string, seats ...string) string {
	c.t.Helper()
	customer := uuid.NewString()
	return c.checkout(c.hold(showID, customer, seats...), customer, nil)
}

// checkout places an order for a hold, with optional test-only headers
// (fakegateway's X-Fake-Payment-Mode), and returns the order id.
func (c *client) checkout(h holdJSON, customer string, headers map[string]string) string {
	c.t.Helper()
	var placed struct{ ID string }
	mustStatus(c.t, c.doWith(http.MethodPost, "/orders", map[string]any{
		"holdId": h.HoldID, "customerId": customer, "contactEmail": "ana@example.com",
	}, headers), http.StatusCreated).decode(c.t, &placed)
	return placed.ID
}

// seatStates reads every seat's state from Ticketing's API.
func (c *client) seatStates(showID string) map[string]string {
	c.t.Helper()
	var seats []struct {
		Ref   string `json:"ref"`
		State string `json:"state"`
	}
	mustStatus(c.t, c.do(http.MethodGet, "/shows/"+showID+"/seats", nil), http.StatusOK).decode(c.t, &seats)
	states := make(map[string]string, len(seats))
	for _, s := range seats {
		states[s.Ref] = s.State
	}
	return states
}

// order reads an order.
func (c *client) order(id string) orderJSON {
	c.t.Helper()
	var o orderJSON
	mustStatus(c.t, c.do(http.MethodGet, "/orders/"+id, nil), http.StatusOK).decode(c.t, &o)
	return o
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
	// TKT-15: EUR 90.00 of seats + 10% fee + 6% VAT in Portugal.
	if order.Total.Amount != 10494 || order.Total.Currency != "EUR" {
		t.Errorf("total = %+v, want EUR 104.94", order.Total)
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
