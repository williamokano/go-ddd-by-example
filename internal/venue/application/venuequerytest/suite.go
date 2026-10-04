// Package venuequerytest is the contract of application.VenueQueries, run
// against every implementation (memory and Postgres).
package venuequerytest

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

var now = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

// Factory returns a repository to seed venues with and the queries that read
// what it stored.
type Factory func(t *testing.T) (application.VenueRepository, application.VenueQueries)

// Run runs the contract. Venues get fresh random IDs, so implementations may
// share storage between tests; List assertions only look at their own venues.
func Run(t *testing.T, newQueries Factory) {
	t.Helper()

	t.Run("get returns a flat view with the capacity precomputed", func(t *testing.T) {
		repo, queries := newQueries(t)
		venue := draftVenue(t, "Coliseu dos Recreios")
		addSections(t, venue)
		save(t, repo, venue)

		got, err := queries.Get(context.Background(), venue.ID())

		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		want := application.VenueView{
			ID: venue.ID().String(), Name: "Coliseu dos Recreios",
			Street: "Rua Portas de Santo Antão 96", City: "Lisboa", Country: "PT",
			Status: "draft", Capacity: 522,
			Sections: []application.SectionView{
				{Code: "ORCH", Name: "Orchestra", Kind: "seated", Capacity: 22, Rows: []application.RowView{{Label: "A", Seats: 10}, {Label: "B", Seats: 12}}},
				{Code: "FLOOR", Name: "Floor", Kind: "ga", Capacity: 500},
			},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("VenueView mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("view capacity matches the domain's Capacity (VEN-7)", func(t *testing.T) {
		repo, queries := newQueries(t)
		venue := draftVenue(t, "Coliseu")
		addSections(t, venue)
		save(t, repo, venue)

		got, err := queries.Get(context.Background(), venue.ID())

		if err != nil {
			t.Fatal(err)
		}
		if got.Capacity != venue.Capacity() {
			t.Errorf("view Capacity = %d, domain Capacity() = %d", got.Capacity, venue.Capacity())
		}
	})

	t.Run("a venue without sections has capacity 0 and no sections", func(t *testing.T) {
		repo, queries := newQueries(t)
		venue := draftVenue(t, "Empty")
		save(t, repo, venue)

		got, err := queries.Get(context.Background(), venue.ID())

		if err != nil {
			t.Fatal(err)
		}
		if got.Capacity != 0 || len(got.Sections) != 0 {
			t.Errorf("Capacity = %d, Sections = %v; want 0 and none", got.Capacity, got.Sections)
		}
	})

	t.Run("get of an unknown id is ErrVenueNotFound", func(t *testing.T) {
		_, queries := newQueries(t)

		_, err := queries.Get(context.Background(), domain.NewVenueID(uuid.New()))

		if !errors.Is(err, application.ErrVenueNotFound) {
			t.Errorf("Get() error = %v, want %v", err, application.ErrVenueNotFound)
		}
	})

	t.Run("list filters by status", func(t *testing.T) {
		repo, queries := newQueries(t)
		draft := draftVenue(t, "A draft")
		active := draftVenue(t, "An active")
		addSections(t, active)
		if err := active.Activate(now); err != nil {
			t.Fatal(err)
		}
		save(t, repo, draft)
		save(t, repo, active)

		activeIDs := listIDs(t, queries, "active")
		draftIDs := listIDs(t, queries, "draft")

		if !slices.Contains(activeIDs, active.ID().String()) || slices.Contains(activeIDs, draft.ID().String()) {
			t.Errorf("List(active) = %v, want %s and not %s", activeIDs, active.ID(), draft.ID())
		}
		if !slices.Contains(draftIDs, draft.ID().String()) || slices.Contains(draftIDs, active.ID().String()) {
			t.Errorf("List(draft) = %v, want %s and not %s", draftIDs, draft.ID(), active.ID())
		}
	})

	t.Run("list views carry their sections and capacity", func(t *testing.T) {
		repo, queries := newQueries(t)
		venue := draftVenue(t, "Listed")
		addSections(t, venue)
		save(t, repo, venue)

		views, err := queries.List(context.Background(), "draft")

		if err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(views, func(v application.VenueView) bool { return v.ID == venue.ID().String() })
		if i < 0 {
			t.Fatalf("venue %s not listed", venue.ID())
		}
		if views[i].Capacity != 522 || len(views[i].Sections) != 2 {
			t.Errorf("listed view = %+v, want capacity 522 and 2 sections", views[i])
		}
	})
}

func draftVenue(t *testing.T, name string) *domain.Venue {
	t.Helper()
	addr, err := domain.NewAddress("Rua Portas de Santo Antão 96", "Lisboa", "PT")
	if err != nil {
		t.Fatal(err)
	}
	venue, err := domain.RegisterVenue(domain.NewVenueID(uuid.New()), name, addr, now)
	if err != nil {
		t.Fatal(err)
	}
	return venue
}

// addSections adds ORCH (rows A=10, B=12) and FLOOR (GA 500).
func addSections(t *testing.T, v *domain.Venue) {
	t.Helper()
	orchCode, _ := domain.NewSectionCode("ORCH")
	a, _ := domain.NewRow("A", 10)
	b, _ := domain.NewRow("B", 12)
	orch, err := domain.NewSeatedSection(orchCode, "Orchestra", []domain.Row{a, b})
	if err != nil {
		t.Fatal(err)
	}
	floorCode, _ := domain.NewSectionCode("FLOOR")
	floor, err := domain.NewGeneralAdmissionSection(floorCode, "Floor", 500)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []domain.Section{orch, floor} {
		if err := v.AddSection(s, now); err != nil {
			t.Fatal(err)
		}
	}
}

func save(t *testing.T, repo application.VenueRepository, v *domain.Venue) {
	t.Helper()
	if err := repo.Save(context.Background(), v); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func listIDs(t *testing.T, queries application.VenueQueries, status string) []string {
	t.Helper()
	views, err := queries.List(context.Background(), status)
	if err != nil {
		t.Fatalf("List(%q) error = %v", status, err)
	}
	ids := make([]string, 0, len(views))
	for _, v := range views {
		ids = append(ids, v.ID)
	}
	return ids
}
