package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/clock"
	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

var fixedNow = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

type fixture struct {
	ctx     context.Context
	shows   *memory.ShowRepository
	layouts *memory.VenueLayouts
	clock   *clock.Fixed
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	return &fixture{
		ctx: context.Background(), shows: memory.NewShowRepository(),
		layouts: memory.NewVenueLayouts(), clock: clock.NewFixed(fixedNow),
	}
}

// activated is the command the consumer builds from venue.activated.v1.
func activated(venueID string) application.OnVenueActivated {
	return application.OnVenueActivated{
		VenueID: venueID, Name: "Coliseu",
		Sections: []application.LayoutSectionSpec{
			{Code: "ORCH", Kind: "seated", Rows: []application.RowSpec{{Label: "A", Seats: 10}}},
			{Code: "FLOOR", Kind: "ga", Capacity: 500},
		},
	}
}

// knownVenue makes Show learn about an active venue, as Kafka would.
func (f *fixture) knownVenue(t *testing.T) domain.VenueID {
	t.Helper()
	id := uuid.NewString()
	if err := application.NewOnVenueActivatedHandler(f.layouts).Handle(f.ctx, activated(id)); err != nil {
		t.Fatal(err)
	}
	venueID, _ := domain.ParseVenueID(id)
	return venueID
}

// showAt saves a show at the venue in the given state, starting `in` from now.
func (f *fixture) showAt(t *testing.T, venueID domain.VenueID, status domain.Status, in time.Duration) domain.ShowID {
	t.Helper()
	start := fixedNow.Add(in)
	schedule, err := domain.NewSchedule(start.Add(-time.Hour), start, start.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	show := domain.RehydrateShow(domain.ShowState{
		ID: domain.NewShowID(uuid.New()), VenueID: venueID, PromoterID: domain.NewPromoterID(uuid.New()),
		Title: "Fado", Schedule: schedule, Status: status,
	})
	if err := f.shows.Save(f.ctx, show); err != nil {
		t.Fatal(err)
	}
	return show.ID()
}

func (f *fixture) status(t *testing.T, id domain.ShowID) domain.Status {
	t.Helper()
	s, err := f.shows.Get(f.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return s.Status()
}
