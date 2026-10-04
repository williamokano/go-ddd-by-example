package domain_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestRegisterVenue_RecordsVenueRegistered(t *testing.T) {
	venue, err := domain.RegisterVenue(aVenueID(), "Coliseu dos Recreios", mustAddress(t), fixedNow)
	if err != nil {
		t.Fatal(err)
	}

	got := venue.PullEvents()

	want := []domain.DomainEvent{
		domain.VenueRegistered{VenueID: aVenueID(), Name: "Coliseu dos Recreios", At: fixedNow},
	}
	if diff := cmp.Diff(want, got, domainValues); diff != "" {
		t.Errorf("events mismatch (-want +got):\n%s", diff)
	}
}

func TestVenue_AddSection_RecordsSectionAdded(t *testing.T) {
	venue := newDraftVenue(t)
	venue.PullEvents() // discard the creation event
	section := seatedSection(t, "ORCH", mustRow(t, "A", 20))

	if err := venue.AddSection(section, fixedNow); err != nil {
		t.Fatal(err)
	}

	want := []domain.DomainEvent{
		domain.SectionAdded{VenueID: venue.ID(), Section: section, At: fixedNow},
	}
	if diff := cmp.Diff(want, venue.PullEvents(), domainValues); diff != "" {
		t.Errorf("events mismatch (-want +got):\n%s", diff)
	}
}

func TestVenue_Activate_RecordsVenueActivatedWithTheLayout(t *testing.T) {
	orch := seatedSection(t, "ORCH", mustRow(t, "A", 10), mustRow(t, "B", 12))
	floor := gaSection(t, "FLOOR", 500)
	venue := newDraftVenue(t, withSection(orch), withSection(floor))
	venue.PullEvents()

	if err := venue.Activate(fixedNow); err != nil {
		t.Fatal(err)
	}

	want := []domain.DomainEvent{domain.VenueActivated{
		VenueID:  venue.ID(),
		Name:     venue.Name(),
		Sections: []domain.Section{orch, floor},
		At:       fixedNow,
	}}
	if diff := cmp.Diff(want, venue.PullEvents(), domainValues); diff != "" {
		t.Errorf("events mismatch (-want +got):\n%s", diff)
	}
}
