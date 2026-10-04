// Package showrepotest is the contract of application.ShowRepository, run
// against the in-memory fake and Postgres.
package showrepotest

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

var now = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

// Run runs the contract. Shows and venues get fresh random IDs.
func Run(t *testing.T, newRepo func(t *testing.T) application.ShowRepository) {
	t.Helper()

	t.Run("get of an unknown id is ErrShowNotFound", func(t *testing.T) {
		_, err := newRepo(t).Get(context.Background(), domain.NewShowID(uuid.New()))

		if !errors.Is(err, application.ErrShowNotFound) {
			t.Errorf("Get() error = %v, want %v", err, application.ErrShowNotFound)
		}
	})

	t.Run("save then get round-trips a draft", func(t *testing.T) {
		repo := newRepo(t)
		show := draft(t, layout(), 0)

		save(t, repo, show)

		assertSameState(t, show, get(t, repo, show.ID()))
	})

	t.Run("save then get round-trips a published show with prices", func(t *testing.T) {
		repo := newRepo(t)
		show := draft(t, layout(), 0)
		if err := show.Price(prices(t), layout(), now); err != nil {
			t.Fatal(err)
		}
		if err := show.Publish(layout(), now); err != nil {
			t.Fatal(err)
		}

		save(t, repo, show)

		assertSameState(t, show, get(t, repo, show.ID()))
	})

	t.Run("an update round-trips, including the cancellation reason", func(t *testing.T) {
		repo := newRepo(t)
		show := draft(t, layout(), 0)
		save(t, repo, show)
		loaded := get(t, repo, show.ID())
		reason, _ := domain.NewCancellationReason("promoter ill")
		if err := loaded.Cancel(reason, now); err != nil {
			t.Fatal(err)
		}

		save(t, repo, loaded)

		assertSameState(t, loaded, get(t, repo, show.ID()))
	})

	t.Run("changes without save are not visible", func(t *testing.T) {
		repo := newRepo(t)
		show := draft(t, layout(), 0)
		save(t, repo, show)
		loaded := get(t, repo, show.ID())
		if err := loaded.Price(prices(t), layout(), now); err != nil {
			t.Fatal(err)
		}

		if !get(t, repo, show.ID()).Prices().IsZero() {
			t.Error("unsaved price list is visible")
		}
	})

	t.Run("save from a stale version is ErrConcurrentModification", func(t *testing.T) {
		repo := newRepo(t)
		show := draft(t, layout(), 0)
		save(t, repo, show)
		first, second := get(t, repo, show.ID()), get(t, repo, show.ID())
		save(t, repo, first)

		if err := repo.Save(context.Background(), second); !errors.Is(err, application.ErrConcurrentModification) {
			t.Errorf("Save() error = %v, want %v", err, application.ErrConcurrentModification)
		}
	})

	t.Run("of two concurrent saves from the same version, exactly one wins", func(t *testing.T) {
		repo := newRepo(t)
		show := draft(t, layout(), 0)
		save(t, repo, show)
		copies := []*domain.Show{get(t, repo, show.ID()), get(t, repo, show.ID())}
		errs := make([]error, 2)
		var wg sync.WaitGroup
		for i, s := range copies {
			wg.Go(func() { errs[i] = repo.Save(context.Background(), s) })
		}
		wg.Wait()

		if (errs[0] == nil) == (errs[1] == nil) {
			t.Errorf("errors = %v; want exactly one success", errs)
		}
	})

	t.Run("save drains the show's events", func(t *testing.T) {
		repo := newRepo(t)
		show := draft(t, layout(), 0)

		save(t, repo, show)

		if left := show.PullEvents(); len(left) != 0 {
			t.Errorf("%d events left after Save", len(left))
		}
	})

	t.Run("list open at venue skips other venues, cancelled and completed shows", func(t *testing.T) {
		repo := newRepo(t)
		venue := layout()
		open := draft(t, venue, 0)
		cancelled := draft(t, venue, 24*time.Hour)
		reason, _ := domain.NewCancellationReason("rain")
		if err := cancelled.Cancel(reason, now); err != nil {
			t.Fatal(err)
		}
		elsewhere := draft(t, layout(), 0)
		for _, s := range []*domain.Show{open, cancelled, elsewhere} {
			save(t, repo, s)
		}

		got, err := repo.ListOpenAtVenue(context.Background(), venue.VenueID)

		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID() != open.ID() {
			t.Errorf("ListOpenAtVenue() = %d shows, want only %s", len(got), open.ID())
		}
	})
}

func layout() domain.VenueLayout {
	return domain.VenueLayout{
		VenueID: domain.NewVenueID(uuid.New()), Name: "Coliseu", Active: true,
		Sections: []domain.LayoutSection{
			{Code: "ORCH", Kind: "seated", Rows: []domain.LayoutRow{{Label: "A", Seats: 10}}},
			{Code: "FLOOR", Kind: "ga", Capacity: 500},
		},
	}
}

func draft(t *testing.T, venue domain.VenueLayout, offset time.Duration) *domain.Show {
	t.Helper()
	start := now.Add(30*24*time.Hour + offset)
	schedule, err := domain.NewSchedule(start.Add(-time.Hour), start, start.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	s, err := domain.DraftShow(domain.NewShowID(uuid.New()), venue, domain.NewPromoterID(uuid.New()), "Fado Night", schedule, now)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func prices(t *testing.T) domain.PriceList {
	t.Helper()
	eur, _ := sharedkernel.NewCurrency("EUR")
	orch, _ := sharedkernel.NewMoney(4500, eur)
	floor, _ := sharedkernel.NewMoney(2500, eur)
	pl, err := domain.NewPriceList(map[string]sharedkernel.Money{"ORCH": orch, "FLOOR": floor})
	if err != nil {
		t.Fatal(err)
	}
	return pl
}

func save(t *testing.T, repo application.ShowRepository, s *domain.Show) {
	t.Helper()
	if err := repo.Save(context.Background(), s); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func get(t *testing.T, repo application.ShowRepository, id domain.ShowID) *domain.Show {
	t.Helper()
	s, err := repo.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	return s
}

func assertSameState(t *testing.T, want, got *domain.Show) {
	t.Helper()
	opts := cmp.AllowUnexported(domain.ShowID{}, domain.VenueID{}, domain.PromoterID{}, domain.Schedule{},
		domain.PriceList{}, sharedkernel.Money{}, sharedkernel.Currency{}, domain.CancellationReason{})
	if diff := cmp.Diff(StateOf(want), StateOf(got), opts); diff != "" {
		t.Errorf("loaded show mismatch (-saved +loaded):\n%s", diff)
	}
}

// StateOf reads a show back through its getters, without the version.
func StateOf(s *domain.Show) domain.ShowState {
	return domain.ShowState{
		ID: s.ID(), VenueID: s.VenueID(), PromoterID: s.PromoterID(), Title: s.Title(),
		Schedule: s.Schedule(), Prices: s.Prices(), Status: s.Status(), CancellationReason: s.CancellationReason(),
	}
}
