package application_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

func TestOnVenueActivated(t *testing.T) {
	t.Run("stores an active layout DraftShow can use", func(t *testing.T) {
		f := newFixture(t)
		id := uuid.NewString()

		err := application.NewOnVenueActivatedHandler(f.layouts).Handle(f.ctx, activated(id))

		if err != nil {
			t.Fatal(err)
		}
		venueID, _ := domain.ParseVenueID(id)
		layout, err := f.layouts.Get(f.ctx, venueID)
		if err != nil {
			t.Fatal(err)
		}
		if !layout.Active || cmp.Diff([]string{"ORCH", "FLOOR"}, layout.SectionCodes()) != "" {
			t.Errorf("layout = %+v", layout)
		}
	})

	t.Run("receiving the same event twice leaves the same state", func(t *testing.T) {
		f := newFixture(t)
		id := uuid.NewString()
		handler := application.NewOnVenueActivatedHandler(f.layouts)
		venueID, _ := domain.ParseVenueID(id)

		_ = handler.Handle(f.ctx, activated(id))
		once, _ := f.layouts.Get(f.ctx, venueID)
		_ = handler.Handle(f.ctx, activated(id))
		twice, _ := f.layouts.Get(f.ctx, venueID)

		if diff := cmp.Diff(once, twice, cmp.AllowUnexported(domain.VenueID{})); diff != "" {
			t.Errorf("a duplicate changed the projection:\n%s", diff)
		}
	})
}

func TestOnVenueRetired(t *testing.T) {
	f := newFixture(t)
	venue := f.knownVenue(t)
	future := f.showAt(t, venue, domain.Published, 30*24*time.Hour)
	futureDraft := f.showAt(t, venue, domain.Draft, 40*24*time.Hour)
	alreadyStarted := f.showAt(t, venue, domain.Published, -30*time.Minute)
	otherVenue := f.showAt(t, f.knownVenue(t), domain.Published, 30*24*time.Hour)

	err := application.NewOnVenueRetiredHandler(f.layouts, f.shows, f.clock).
		Handle(f.ctx, application.OnVenueRetired{VenueID: venue.String()})

	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[domain.ShowID]domain.Status{
		future: domain.Cancelled, futureDraft: domain.Cancelled,
		alreadyStarted: domain.Published, otherVenue: domain.Published,
	} {
		if got := f.status(t, id); got != want {
			t.Errorf("show %s status = %v, want %v (SHW-7)", id, got, want)
		}
	}
	layout, _ := f.layouts.Get(f.ctx, venue)
	if layout.Active {
		t.Error("the retired venue is still active in the projection")
	}
	cancelled := 0
	for _, ev := range f.shows.Published() {
		if c, ok := ev.(domain.ShowCancelled); ok && c.Reason.String() == domain.ReasonVenueRetired {
			cancelled++
		}
	}
	if cancelled != 2 {
		t.Errorf("%d ShowCancelled{venue_retired} events, want 2", cancelled)
	}
}

func TestOnVenueRetired_UnknownVenue(t *testing.T) {
	f := newFixture(t)
	id := uuid.NewString()

	err := application.NewOnVenueRetiredHandler(f.layouts, f.shows, f.clock).Handle(f.ctx, application.OnVenueRetired{VenueID: id})

	if err != nil {
		t.Fatalf("error = %v, want the retirement remembered", err)
	}
	venueID, _ := domain.ParseVenueID(id)
	if layout, err := f.layouts.Get(f.ctx, venueID); err != nil || layout.Active {
		t.Errorf("layout = %+v, %v; want a known, inactive venue", layout, err)
	}
}
