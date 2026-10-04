package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/clock"
	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/ids"
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

	draft    *application.DraftShowHandler
	price    *application.PriceShowHandler
	publish  *application.PublishShowHandler
	cancel   *application.CancelShowHandler
	soldOut  *application.MarkShowSoldOutHandler
	promoter string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	shows, layouts, clk := memory.NewShowRepository(), memory.NewVenueLayouts(), clock.NewFixed(fixedNow)
	return &fixture{
		ctx: context.Background(), shows: shows, layouts: layouts, clock: clk,
		draft:    application.NewDraftShowHandler(shows, layouts, ids.NewShowIDs(idgen.NewSequence()), clk),
		price:    application.NewPriceShowHandler(shows, layouts, clk),
		publish:  application.NewPublishShowHandler(shows, layouts, clk),
		cancel:   application.NewCancelShowHandler(shows, clk),
		soldOut:  application.NewMarkShowSoldOutHandler(shows, clk),
		promoter: uuid.NewString(),
	}
}

// draftCmd drafts a show at venue, starting `in` from now, for f.promoter.
func (f *fixture) draftCmd(venue domain.VenueID, in time.Duration) application.DraftShow {
	start := fixedNow.Add(in)
	return application.DraftShow{
		PromoterID: f.promoter, VenueID: venue.String(), Title: "Fado Night",
		DoorsOpen: start.Add(-time.Hour), StartsAt: start, EndsAt: start.Add(2 * time.Hour),
	}
}

func (f *fixture) fullPrices(showID domain.ShowID) application.PriceShow {
	return application.PriceShow{ShowID: showID.String(), PromoterID: f.promoter, Prices: []application.PriceSpec{
		{Section: "ORCH", Amount: 4500, Currency: "EUR"}, {Section: "FLOOR", Amount: 2500, Currency: "EUR"},
	}}
}

// draftedShow drafts a show at a fresh known venue.
func (f *fixture) draftedShow(t *testing.T) domain.ShowID {
	t.Helper()
	id, err := f.draft.Handle(f.ctx, f.draftCmd(f.knownVenue(t), 30*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *fixture) publishedShow(t *testing.T) domain.ShowID {
	t.Helper()
	id := f.draftedShow(t)
	if err := f.price.Handle(f.ctx, f.fullPrices(id)); err != nil {
		t.Fatal(err)
	}
	if err := f.publish.Handle(f.ctx, application.PublishShow{ShowID: id.String(), PromoterID: f.promoter}); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *fixture) lastEvent(t *testing.T) sharedkernel.DomainEvent {
	t.Helper()
	events := f.shows.Published()
	if len(events) == 0 {
		t.Fatal("no events")
	}
	return events[len(events)-1]
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
