package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/platform/clock"
	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/ids"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

var fixedNow = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

// fixture wires every Venue use case to in-memory fakes. Tests use the
// handlers both to arrange (Given) and to act (When).
type fixture struct {
	ctx      context.Context
	repo     *memory.VenueRepository
	clock    *clock.Fixed
	register *application.RegisterVenueHandler
	add      *application.AddSectionHandler
	activate *application.ActivateVenueHandler
	retire   *application.RetireVenueHandler
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	repo := memory.NewVenueRepository()
	clk := clock.NewFixed(fixedNow)
	return &fixture{
		ctx:      context.Background(),
		repo:     repo,
		clock:    clk,
		register: application.NewRegisterVenueHandler(repo, ids.NewVenueIDs(idgen.NewSequence()), clk),
		add:      application.NewAddSectionHandler(repo, clk),
		activate: application.NewActivateVenueHandler(repo, clk),
		retire:   application.NewRetireVenueHandler(repo, clk),
	}
}

func validRegisterVenue() application.RegisterVenue {
	return application.RegisterVenue{
		Name: "Coliseu dos Recreios", Street: "Rua Portas de Santo Antão 96", City: "Lisboa", Country: "PT",
	}
}

// draftVenue registers a draft venue with one GA section, "FLOOR".
func (f *fixture) draftVenue(t *testing.T) domain.VenueID {
	t.Helper()
	id, err := f.register.Handle(f.ctx, validRegisterVenue())
	if err != nil {
		t.Fatal(err)
	}
	if err := f.add.Handle(f.ctx, application.AddSection{VenueID: id.String(), Code: "FLOOR", Kind: application.KindGA, Capacity: 100}); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *fixture) activeVenue(t *testing.T) domain.VenueID {
	t.Helper()
	id := f.draftVenue(t)
	if err := f.activate.Handle(f.ctx, application.ActivateVenue{VenueID: id.String()}); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *fixture) venue(t *testing.T, id domain.VenueID) *domain.Venue {
	t.Helper()
	v, err := f.repo.Get(f.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func (f *fixture) lastEvent(t *testing.T) domain.DomainEvent {
	t.Helper()
	events := f.repo.Published()
	if len(events) == 0 {
		t.Fatal("no events published")
	}
	return events[len(events)-1]
}
