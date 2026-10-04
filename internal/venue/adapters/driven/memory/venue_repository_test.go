package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application/venuerepotest"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestVenueRepository_Contract(t *testing.T) {
	venuerepotest.Run(t, func(*testing.T) application.VenueRepository { return memory.NewVenueRepository() })
}

// Published is specific to the fake: it stands in for the outbox.
func TestVenueRepository_Published(t *testing.T) {
	repo := memory.NewVenueRepository()
	venue := newDraftVenue(t)

	if err := repo.Save(context.Background(), venue); err != nil {
		t.Fatal(err)
	}

	want := []domain.DomainEvent{
		domain.VenueRegistered{VenueID: venue.ID(), Name: venue.Name(), At: fixedNow},
	}
	if diff := cmp.Diff(want, repo.Published(), cmp.AllowUnexported(domain.VenueID{})); diff != "" {
		t.Errorf("Published() mismatch (-want +got):\n%s", diff)
	}
}

var fixedNow = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

func newDraftVenue(t *testing.T) *domain.Venue {
	t.Helper()
	addr, err := domain.NewAddress("Rua Portas de Santo Antão 96", "Lisboa", "PT")
	if err != nil {
		t.Fatal(err)
	}
	venue, err := domain.RegisterVenue(domain.NewVenueID(uuid.New()), "Coliseu dos Recreios", addr, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	return venue
}
