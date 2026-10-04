//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"
)

// The wiring, end to end: HTTP → use case → Postgres → HTTP.
func TestVenue_LayOutAndActivate(t *testing.T) {
	c := newClient(t)
	id := c.registerVenue("Coliseu dos Recreios")

	mustStatus(t, c.do(http.MethodPost, "/venues/"+id+"/sections", map[string]any{
		"code": "ORCH", "name": "Orchestra", "kind": "seated",
		"rows": []map[string]any{{"label": "A", "seats": 10}, {"label": "B", "seats": 12}},
	}), http.StatusNoContent)
	mustStatus(t, c.do(http.MethodPost, "/venues/"+id+"/sections", map[string]any{
		"code": "FLOOR", "name": "Floor", "kind": "ga", "capacity": 500,
	}), http.StatusNoContent)
	mustStatus(t, c.do(http.MethodPost, "/venues/"+id+"/activation", nil), http.StatusNoContent)

	var venue venueJSON
	mustStatus(t, c.do(http.MethodGet, "/venues/"+id, nil), http.StatusOK).decode(t, &venue)
	if venue.Status != "active" || venue.Capacity != 522 {
		t.Errorf("venue = %+v, want active with capacity 522", venue)
	}
}

func TestVenue_ActivateWithoutSections(t *testing.T) {
	c := newClient(t)
	id := c.registerVenue("Empty Hall")

	var problem problemJSON
	mustStatus(t, c.do(http.MethodPost, "/venues/"+id+"/activation", nil), http.StatusConflict).decode(t, &problem)

	if problem.Title != "Venue has no sections" {
		t.Errorf("title = %q, want %q (VEN-5)", problem.Title, "Venue has no sections")
	}
}
