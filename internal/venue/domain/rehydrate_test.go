package domain_test

import (
	"errors"
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

	t.Run("records no events: nothing new happened", func(t *testing.T) {
		venue := domain.RehydrateVenue(activeVenueState(t))

		if got := venue.PullEvents(); len(got) != 0 {
			t.Errorf("PullEvents() = %v, want none", got)
		}
	})

	t.Run("a rehydrated active venue still enforces VEN-4", func(t *testing.T) {
		venue := domain.RehydrateVenue(activeVenueState(t))

		err := venue.AddSection(gaSection(t, "BALCONY", 100), fixedNow)

		if !errors.Is(err, domain.ErrVenueNotDraft) {
			t.Errorf("AddSection() error = %v, want %v", err, domain.ErrVenueNotDraft)
		}
	})

	t.Run("does not share the state's sections slice", func(t *testing.T) {
		state := activeVenueState(t)
		venue := domain.RehydrateVenue(state)

		state.Sections[0] = gaSection(t, "HACK", 1)

		if diff := cmp.Diff([]string{"ORCH", "FLOOR"}, sectionCodes(venue.Sections())); diff != "" {
			t.Errorf("Sections() changed with the state (-want +got):\n%s", diff)
		}
	})
}

func TestRegisterVenue_StartsAtVersionZero(t *testing.T) {
	venue := newDraftVenue(t)

	if got := venue.Version(); got != 0 {
		t.Errorf("Version() = %d, want 0", got)
	}
}
