package httpapi_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driving/httpapi"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

var venueID = domain.NewVenueID(uuid.MustParse("0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f"))

// stubs is one stub per use case: it records the command it got and returns
// err. A stub, not a fake: the HTTP adapter has no logic worth simulating.
type stubs struct {
	err        error
	registered *application.RegisterVenue
	added      *application.AddSection
	activated  *application.ActivateVenue
	retired    *application.RetireVenue
	view       application.VenueView
	listed     string
}

func (s *stubs) handler() http.Handler {
	return httpapi.Routes(httpapi.UseCases{
		Register: registerStub{s}, AddSection: addSectionStub{s},
		Activate: activateStub{s}, Retire: retireStub{s}, Queries: queriesStub{s},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

type registerStub struct{ s *stubs }

func (r registerStub) Handle(_ context.Context, cmd application.RegisterVenue) (domain.VenueID, error) {
	r.s.registered = &cmd
	return venueID, r.s.err
}

type addSectionStub struct{ s *stubs }

func (a addSectionStub) Handle(_ context.Context, cmd application.AddSection) error {
	a.s.added = &cmd
	return a.s.err
}

type activateStub struct{ s *stubs }

func (a activateStub) Handle(_ context.Context, cmd application.ActivateVenue) error {
	a.s.activated = &cmd
	return a.s.err
}

type retireStub struct{ s *stubs }

func (r retireStub) Handle(_ context.Context, cmd application.RetireVenue) error {
	r.s.retired = &cmd
	return r.s.err
}

type queriesStub struct{ s *stubs }

func (q queriesStub) Get(context.Context, domain.VenueID) (application.VenueView, error) {
	return q.s.view, q.s.err
}

func (q queriesStub) List(_ context.Context, status string) ([]application.VenueView, error) {
	q.s.listed = status
	return []application.VenueView{q.s.view}, q.s.err
}
