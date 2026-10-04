//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The scenarios below need short holds: make test-e2e starts the app with
// HOLD_TTL=3s and HOLD_SWEEP_INTERVAL=500ms (docker-compose.yml defaults to
// the real 10m and 5s).

// S2 — Hold expires: the seat goes back on sale, the hold can no longer be
// bought, and another customer can hold the seat (TKT-4).
func TestS2_HoldExpires(t *testing.T) {
	c := newClient(t)
	show := c.publishedShow()
	customer := uuid.NewString()
	h := c.hold(show, customer, "ORCH/A/1")
	if got := c.seatStates(show)["ORCH/A/1"]; got != "held" {
		t.Fatalf("ORCH/A/1 is %q right after the hold, want held", got)
	}

	eventually(t, 20*time.Second, "the seat is available again", func() bool {
		return c.seatStates(show)["ORCH/A/1"] == "available"
	})
	r := c.do(http.MethodPost, "/orders", map[string]any{
		"holdId": h.HoldID, "customerId": customer, "contactEmail": "ana@example.com",
	})
	if r.Status != http.StatusNotFound && r.Status != http.StatusConflict {
		t.Errorf("checkout of an expired hold: status %d, want 404 or 409: %s", r.Status, r.Body)
	}
	if status := c.rawHold(show, uuid.NewString(), "ORCH/A/1"); status != http.StatusCreated {
		t.Errorf("another customer holds the seat: status %d, want 201", status)
	}
}

// S3 — Late payment: the provider approves after the hold expired, so the
// saga cannot confirm the seats and compensates with a refund (TKT-7).
func TestS3_LatePaymentIsRefunded(t *testing.T) {
	c := newClient(t)
	show := c.publishedShow()
	customer := uuid.NewString()
	h := c.hold(show, customer, "ORCH/A/1")
	order := c.checkout(h, customer, map[string]string{"X-Fake-Payment-Mode": "delay:5s"})

	var got orderJSON
	eventually(t, 30*time.Second, "the late order is refunded", func() bool {
		got = c.order(order)
		return got.Status == "refunded"
	})
	if len(got.Tickets) != 0 {
		t.Errorf("a refunded late order has tickets: %+v", got.Tickets)
	}
	if s := c.seatStates(show)["ORCH/A/1"]; s != "available" {
		t.Errorf("ORCH/A/1 is %q, want available", s)
	}
}

// S5 — Venue retires, the full cascade across three contexts:
// venue.retired.v1 → Show cancels the show → show.cancelled.v1 → Ticketing
// closes the inventory, voids the tickets and refunds the order.
func TestS5_VenueRetirementCascades(t *testing.T) {
	c := newClient(t)
	venue := c.activeVenue()
	show := c.publishedShowAt(venue, uuid.NewString())
	order := c.buy(show, "ORCH/A/1", "ORCH/A/2")
	eventually(t, 20*time.Second, "the order is fulfilled", func() bool {
		return c.order(order).Status == "fulfilled"
	})

	mustStatus(t, c.do(http.MethodPost, "/venues/"+venue+"/retirement", nil), http.StatusNoContent)

	eventually(t, 20*time.Second, "Show cancels the show", func() bool {
		return c.showStatus(show) == "cancelled"
	})
	var got orderJSON
	eventually(t, 20*time.Second, "Ticketing voids the tickets and refunds the order", func() bool {
		got = c.order(order)
		if got.Status != "refunded" || len(got.Tickets) != 2 {
			return false
		}
		for _, tk := range got.Tickets {
			if tk.Status != "voided" {
				return false
			}
		}
		return true
	})
}

// S6 — Race for the last seat: many customers hold the same seat at once;
// exactly one wins, the others get 409, and nobody gets a 500.
func TestS6_RaceForTheLastSeat(t *testing.T) {
	c := newClient(t)
	show := c.publishedShow()
	c.hold(show, uuid.NewString(), "FLOOR/GA/0001") // waits until the inventory is open

	const racers = 20
	statuses := make([]int, racers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := range racers {
		wg.Go(func() {
			<-start
			statuses[i] = c.rawHold(show, uuid.NewString(), "ORCH/A/1")
		})
	}
	close(start)
	wg.Wait()

	counts := map[int]int{}
	for _, s := range statuses {
		counts[s]++
	}
	if counts[http.StatusCreated] != 1 || counts[http.StatusConflict] != racers-1 {
		t.Errorf("statuses = %v, want exactly one 201 and %d × 409", counts, racers-1)
	}
}

// rawHold posts a hold and returns the status, or 0 on a transport error.
// It never calls t.Fatal, so goroutines can use it.
func (c *client) rawHold(showID, customerID string, seats ...string) int {
	b, _ := json.Marshal(map[string]any{"customerId": customerID, "seats": seats})
	req, err := http.NewRequestWithContext(c.t.Context(), http.MethodPost, c.base+"/shows/"+showID+"/holds", bytes.NewReader(b))
	if err != nil {
		return 0
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return 0
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}
