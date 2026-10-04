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
