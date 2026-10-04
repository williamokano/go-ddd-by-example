package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestVenueRepository_Get_UnknownID(t *testing.T) {
	repo := memory.NewVenueRepository()

	_, err := repo.Get(context.Background(), domain.NewVenueID(uuid.New()))

	if !errors.Is(err, application.ErrVenueNotFound) {
		t.Errorf("Get() error = %v, want %v", err, application.ErrVenueNotFound)
	}
}

func TestVenueRepository_SaveThenGet(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewVenueRepository()
	venue := newDraftVenue(t)

	if err := repo.Save(ctx, venue); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := repo.Get(ctx, venue.ID())

	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if diff := cmp.Diff(stateOf(venue), stateOf(got), domainValues, ignoreVersion); diff != "" {
		t.Errorf("loaded venue mismatch (-saved +loaded):\n%s", diff)
	}
}

var (
	fixedNow     = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
	domainValues = cmp.AllowUnexported(
		domain.VenueID{}, domain.Address{}, domain.SectionCode{}, domain.Section{}, domain.Row{},
	)
	ignoreVersion = cmpopts.IgnoreFields(domain.VenueState{}, "Version")
)

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

func stateOf(v *domain.Venue) domain.VenueState {
	return domain.VenueState{
		ID: v.ID(), Name: v.Name(), Address: v.Address(), Status: v.Status(),
		Sections: v.Sections(), Version: v.Version(),
	}
}
