package domain_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func activeVenueState(t *testing.T) domain.VenueState {
	t.Helper()
	return domain.VenueState{
		ID:      aVenueID(),
		Name:    "Coliseu dos Recreios",
		Address: mustAddress(t),
		Status:  domain.Active,
		Sections: []domain.Section{
			seatedSection(t, "ORCH", mustRow(t, "A", 10)),
			gaSection(t, "FLOOR", 500),
		},
		Version: 7,
	}
}

// stateOf reads a venue back through its getters, the way a repository does.
func stateOf(v *domain.Venue) domain.VenueState {
	return domain.VenueState{
		ID:       v.ID(),
		Name:     v.Name(),
		Address:  v.Address(),
		Status:   v.Status(),
		Sections: v.Sections(),
		Version:  v.Version(),
	}
}

func TestRehydrateVenue(t *testing.T) {
	t.Run("exposes exactly the given state", func(t *testing.T) {
		state := activeVenueState(t)

		venue := domain.RehydrateVenue(state)

		if diff := cmp.Diff(state, stateOf(venue), domainValues); diff != "" {
			t.Errorf("state mismatch (-want +got):\n%s", diff)
		}
	})
}
