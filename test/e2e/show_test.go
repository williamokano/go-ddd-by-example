//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

type showJSON struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// activeVenue registers, lays out (ORCH seated + FLOOR GA) and activates a venue.
func (c *client) activeVenue() string {
	c.t.Helper()
	id := c.registerVenue("Coliseu " + uuid.NewString()[:8])
	mustStatus(c.t, c.do(http.MethodPost, "/venues/"+id+"/sections", map[string]any{
		"code": "ORCH", "name": "Orchestra", "kind": "seated",
		"rows": []map[string]any{{"label": "A", "seats": 2}},
	}), http.StatusNoContent)
	mustStatus(c.t, c.do(http.MethodPost, "/venues/"+id+"/sections", map[string]any{
		"code": "FLOOR", "name": "Floor", "kind": "ga", "capacity": 3,
	}), http.StatusNoContent)
	mustStatus(c.t, c.do(http.MethodPost, "/venues/"+id+"/activation", nil), http.StatusNoContent)
	return id
}

// schedule returns a show schedule starting `in` from now, lasting 2 hours.
func schedule(in time.Duration) map[string]any {
	start := time.Now().UTC().Add(in).Truncate(time.Minute)
	return map[string]any{
		"doorsOpen": start.Add(-time.Hour), "startsAt": start, "endsAt": start.Add(2 * time.Hour),
	}
}

// draftShow drafts a show, waiting until Show has learned about the venue
// from Kafka (venue.activated.v1).
func (c *client) draftShow(venueID, promoterID string, when map[string]any) string {
	c.t.Helper()
	body := map[string]any{"promoterId": promoterID, "venueId": venueID, "title": "Fado Night"}
	for k, v := range when {
		body[k] = v
	}
	var created struct{ ID string }
	eventually(c.t, 20*time.Second, "show can be drafted at the new venue", func() bool {
		r := c.do(http.MethodPost, "/shows", body)
		if r.Status == http.StatusCreated {
			r.decode(c.t, &created)
			return true
		}
		return false
	})
	return created.ID
}

func (c *client) priceAndPublish(showID, promoterID string) {
	c.t.Helper()
	mustStatus(c.t, c.do(http.MethodPost, "/shows/"+showID+"/prices", map[string]any{
		"promoterId": promoterID,
		"prices": []map[string]any{
			{"section": "ORCH", "amount": 4500, "currency": "EUR"},
			{"section": "FLOOR", "amount": 2500, "currency": "EUR"},
		},
	}), http.StatusNoContent)
	mustStatus(c.t, c.do(http.MethodPost, "/shows/"+showID+"/publication", map[string]any{"promoterId": promoterID}), http.StatusNoContent)
}

func TestShow_DraftPriceAndPublish(t *testing.T) {
	c := newClient(t)
	promoter := uuid.NewString()
	venue := c.activeVenue()

	show := c.draftShow(venue, promoter, schedule(30*24*time.Hour))
	c.priceAndPublish(show, promoter)

	var got showJSON
	mustStatus(t, c.do(http.MethodGet, "/shows/"+show, nil), http.StatusOK).decode(t, &got)
	if got.Status != "published" {
		t.Errorf("status = %q, want published", got.Status)
	}
}

func TestShow_OverlappingShowsAreRejected(t *testing.T) {
	c := newClient(t)
	promoter := uuid.NewString()
	venue := c.activeVenue()
	c.draftShow(venue, promoter, schedule(30*24*time.Hour))

	overlapping := map[string]any{"promoterId": promoter, "venueId": venue, "title": "Late Fado"}
	for k, v := range schedule(30*24*time.Hour + time.Hour) {
		overlapping[k] = v
	}
	var problem problemJSON
	mustStatus(t, c.do(http.MethodPost, "/shows", overlapping), http.StatusConflict).decode(t, &problem)

	if problem.Title != "Schedule conflict" {
		t.Errorf("title = %q, want %q (SHW-3)", problem.Title, "Schedule conflict")
	}
}
