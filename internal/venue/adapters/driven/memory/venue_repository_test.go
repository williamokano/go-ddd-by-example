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

func TestVenueRepository_ChangesWithoutSaveAreNotVisible(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewVenueRepository()
	venue := newDraftVenue(t)
	if err := repo.Save(ctx, venue); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.Get(ctx, venue.ID())
	if err != nil {
		t.Fatal(err)
	}

	floor, _ := domain.NewSectionCode("FLOOR")
	section, _ := domain.NewGeneralAdmissionSection(floor, "Floor", 500)
	if err := loaded.AddSection(section, fixedNow); err != nil { // mutate, but never Save
		t.Fatal(err)
	}

	again, err := repo.Get(ctx, venue.ID())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(again.Sections()); got != 0 {
		t.Errorf("unsaved change is visible: %d sections, want 0", got)
	}
}

func TestVenueRepository_Save_StaleVersion(t *testing.T) {
	ctx := context.Background()
	repo := memory.NewVenueRepository()
	venue := newDraftVenue(t)
	if err := repo.Save(ctx, venue); err != nil {
		t.Fatal(err)
	}
	first, _ := repo.Get(ctx, venue.ID())
	second, _ := repo.Get(ctx, venue.ID())
	if err := repo.Save(ctx, first); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}

	err := repo.Save(ctx, second)

	if !errors.Is(err, application.ErrConcurrentModification) {
		t.Errorf("second Save() error = %v, want %v", err, application.ErrConcurrentModification)
	}
}

func TestVenueRepository_Save_DrainsEvents(t *testing.T) {
	repo := memory.NewVenueRepository()
	venue := newDraftVenue(t)

	if err := repo.Save(context.Background(), venue); err != nil {
		t.Fatal(err)
	}

	want := []domain.DomainEvent{
		domain.VenueRegistered{VenueID: venue.ID(), Name: venue.Name(), At: fixedNow},
	}
	if diff := cmp.Diff(want, repo.Published(), domainValues); diff != "" {
		t.Errorf("Published() mismatch (-want +got):\n%s", diff)
	}
	if left := venue.PullEvents(); len(left) != 0 {
		t.Errorf("venue still has %d pending events after Save", len(left))
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
