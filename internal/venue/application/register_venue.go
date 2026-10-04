package application

import (
	"context"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// RegisterVenue is the command to register a new venue. Its fields are
// primitives: any driving adapter (HTTP, CLI, test) can build one.
type RegisterVenue struct {
	Name    string
	Street  string
	City    string
	Country string
}

// RegisterVenueHandler is the RegisterVenue use case.
type RegisterVenueHandler struct {
	venues VenueRepository
	ids    IDGenerator
	clock  Clock
}

// NewRegisterVenueHandler wires the use case to its driven ports.
func NewRegisterVenueHandler(venues VenueRepository, ids IDGenerator, clock Clock) *RegisterVenueHandler {
	return &RegisterVenueHandler{venues: venues, ids: ids, clock: clock}
}

// Handle registers the venue and returns its new ID. Errors are wrapped with
// the use case's name; errors.Is still finds the domain error underneath.
func (h *RegisterVenueHandler) Handle(ctx context.Context, cmd RegisterVenue) (domain.VenueID, error) {
	addr, err := domain.NewAddress(cmd.Street, cmd.City, cmd.Country)
	if err != nil {
		return domain.VenueID{}, fmt.Errorf("register venue: %w", err)
	}
	venue, err := domain.RegisterVenue(h.ids.NewVenueID(), cmd.Name, addr, h.clock.Now())
	if err != nil {
		return domain.VenueID{}, fmt.Errorf("register venue: %w", err)
	}
	if err := h.venues.Save(ctx, venue); err != nil {
		return domain.VenueID{}, fmt.Errorf("register venue: %w", err)
	}
	return venue.ID(), nil
}
