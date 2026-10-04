// Package venuerepotest is the contract of application.VenueRepository,
// written once as tests. Every implementation runs it: the in-memory fake
// (fast tests trust it) and Postgres (Part 3). If both pass, the fake is honest.
package venuerepotest

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

var now = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

// Run runs the contract against repositories built by newRepo. Each test uses
// fresh random IDs, so implementations may share storage between tests.
func Run(t *testing.T, newRepo func(t *testing.T) application.VenueRepository) {
	t.Helper()

	t.Run("get of an unknown id is ErrVenueNotFound", func(t *testing.T) {
		_, err := newRepo(t).Get(context.Background(), domain.NewVenueID(uuid.New()))

		if !errors.Is(err, application.ErrVenueNotFound) {
			t.Errorf("Get() error = %v, want %v", err, application.ErrVenueNotFound)
		}
	})

	t.Run("save then get round-trips a draft venue", func(t *testing.T) {
		repo := newRepo(t)
		venue := draftVenue(t)

		mustSave(t, repo, venue)

		assertSameState(t, venue, mustGet(t, repo, venue.ID()))
	})

	t.Run("save then get round-trips an active venue with seated and GA sections", func(t *testing.T) {
		repo := newRepo(t)
		venue := activeVenue(t)

		mustSave(t, repo, venue)

		assertSameState(t, venue, mustGet(t, repo, venue.ID()))
	})

	t.Run("changes without save are not visible", func(t *testing.T) {
		repo := newRepo(t)
		venue := draftVenue(t)
		mustSave(t, repo, venue)
		loaded := mustGet(t, repo, venue.ID())

		if err := loaded.AddSection(gaSection(t, "BOX", 4), now); err != nil {
			t.Fatal(err)
		}

		if got := len(mustGet(t, repo, venue.ID()).Sections()); got != 0 {
			t.Errorf("unsaved change is visible: %d sections, want 0", got)
		}
	})

	t.Run("an update round-trips", func(t *testing.T) {
		repo := newRepo(t)
		venue := draftVenue(t)
		mustSave(t, repo, venue)
		loaded := mustGet(t, repo, venue.ID())
		if err := loaded.AddSection(gaSection(t, "BOX", 4), now); err != nil {
			t.Fatal(err)
		}

		mustSave(t, repo, loaded)

		assertSameState(t, loaded, mustGet(t, repo, venue.ID()))
	})

	t.Run("save from a stale version is ErrConcurrentModification", func(t *testing.T) {
		repo := newRepo(t)
		venue := draftVenue(t)
		mustSave(t, repo, venue)
		first, second := mustGet(t, repo, venue.ID()), mustGet(t, repo, venue.ID())
		mustSave(t, repo, first)

		err := repo.Save(context.Background(), second)

		if !errors.Is(err, application.ErrConcurrentModification) {
			t.Errorf("Save() error = %v, want %v", err, application.ErrConcurrentModification)
		}
	})

	t.Run("registering the same id twice is ErrConcurrentModification", func(t *testing.T) {
		repo := newRepo(t)
		venue := draftVenue(t)
		mustSave(t, repo, venue)
		twin, err := domain.RegisterVenue(venue.ID(), "Twin", venue.Address(), now)
		if err != nil {
			t.Fatal(err)
		}

		err = repo.Save(context.Background(), twin)

		if !errors.Is(err, application.ErrConcurrentModification) {
			t.Errorf("Save() error = %v, want %v", err, application.ErrConcurrentModification)
		}
	})

	t.Run("of two concurrent saves from the same version, exactly one wins", func(t *testing.T) {
		repo := newRepo(t)
		venue := draftVenue(t)
		mustSave(t, repo, venue)
		copies := []*domain.Venue{mustGet(t, repo, venue.ID()), mustGet(t, repo, venue.ID())}

		errs := make([]error, len(copies))
		var wg sync.WaitGroup
		for i, v := range copies {
			wg.Go(func() { errs[i] = repo.Save(context.Background(), v) })
		}
		wg.Wait()

		var wins, conflicts int
		for _, err := range errs {
			switch {
			case err == nil:
				wins++
			case errors.Is(err, application.ErrConcurrentModification):
				conflicts++
			default:
				t.Errorf("Save() unexpected error = %v", err)
			}
		}
		if wins != 1 || conflicts != 1 {
			t.Errorf("wins = %d, conflicts = %d; want exactly one of each", wins, conflicts)
		}
	})

	t.Run("save drains the venue's pending events", func(t *testing.T) {
		repo := newRepo(t)
		venue := draftVenue(t)

		mustSave(t, repo, venue)

		if left := venue.PullEvents(); len(left) != 0 {
			t.Errorf("venue still has %d pending events after Save", len(left))
		}
	})

	t.Run("a loaded venue has no pending events", func(t *testing.T) {
		repo := newRepo(t)
		venue := draftVenue(t)
		mustSave(t, repo, venue)

		if got := mustGet(t, repo, venue.ID()).PullEvents(); len(got) != 0 {
			t.Errorf("loaded venue has events %v, want none", got)
		}
	})
}

func draftVenue(t *testing.T) *domain.Venue {
	t.Helper()
	addr, err := domain.NewAddress("Rua Portas de Santo Antão 96", "Lisboa", "PT")
	if err != nil {
		t.Fatal(err)
	}
	venue, err := domain.RegisterVenue(domain.NewVenueID(uuid.New()), "Coliseu dos Recreios", addr, now)
	if err != nil {
		t.Fatal(err)
	}
	return venue
}

func activeVenue(t *testing.T) *domain.Venue {
	t.Helper()
	venue := draftVenue(t)
	code, err := domain.NewSectionCode("ORCH")
	if err != nil {
		t.Fatal(err)
	}
	a, _ := domain.NewRow("A", 10)
	b, _ := domain.NewRow("B", 12)
	orch, err := domain.NewSeatedSection(code, "Orchestra", []domain.Row{a, b})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []domain.Section{orch, gaSection(t, "FLOOR", 500)} {
		if err := venue.AddSection(s, now); err != nil {
			t.Fatal(err)
		}
	}
	if err := venue.Activate(now); err != nil {
		t.Fatal(err)
	}
	return venue
}

func gaSection(t *testing.T, code string, capacity int) domain.Section {
	t.Helper()
	c, err := domain.NewSectionCode(code)
	if err != nil {
		t.Fatal(err)
	}
	s, err := domain.NewGeneralAdmissionSection(c, code, capacity)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func mustSave(t *testing.T, repo application.VenueRepository, v *domain.Venue) {
	t.Helper()
	if err := repo.Save(context.Background(), v); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func mustGet(t *testing.T, repo application.VenueRepository, id domain.VenueID) *domain.Venue {
	t.Helper()
	v, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	return v
}

// assertSameState compares everything but the version, which the repository
// owns.
func assertSameState(t *testing.T, want, got *domain.Venue) {
	t.Helper()
	opts := cmp.AllowUnexported(domain.VenueID{}, domain.Address{}, domain.SectionCode{}, domain.Section{}, domain.Row{})
	if diff := cmp.Diff(stateOf(want), stateOf(got), opts); diff != "" {
		t.Errorf("loaded venue mismatch (-saved +loaded):\n%s", diff)
	}
}

func stateOf(v *domain.Venue) domain.VenueState {
	return domain.VenueState{
		ID: v.ID(), Name: v.Name(), Address: v.Address(), Status: v.Status(), Sections: v.Sections(),
	}
}
