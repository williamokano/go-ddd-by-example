//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"
	"time"
)

// 9.3: accessibility travels Venue → venue.activated.v1 (additive) → Show →
// show.published.v2 → Ticketing's seat map.
func TestAccessibleSeats_ReachTheSeatMap(t *testing.T) {
	c := newClient(t)
	show := c.publishedShow() // ORCH/A/2 is accessible

	var seats []struct {
		Ref        string `json:"ref"`
		Accessible bool   `json:"accessible"`
	}
	eventually(t, 20*time.Second, "the inventory opens", func() bool {
		r := c.do(http.MethodGet, "/shows/"+show+"/seats", nil)
		if r.Status != http.StatusOK {
			return false
		}
		r.decode(t, &seats)
		return true
	})
	for _, s := range seats {
		if s.Accessible != (s.Ref == "ORCH/A/2") {
			t.Errorf("%s accessible = %v", s.Ref, s.Accessible)
		}
	}
}
