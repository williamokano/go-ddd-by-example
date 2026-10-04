package application

import (
	"context"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// ActivateVenue is the command to open a draft venue for shows.
type ActivateVenue struct{ VenueID string }

// ActivateVenueHandler is the ActivateVenue use case.
type ActivateVenueHandler struct {
	venues VenueRepository
	clock  Clock
}

// NewActivateVenueHandler wires the use case to its driven ports.
func NewActivateVenueHandler(venues VenueRepository, clock Clock) *ActivateVenueHandler {
	return &ActivateVenueHandler{venues: venues, clock: clock}
}

// Handle activates the venue.
func (h *ActivateVenueHandler) Handle(ctx context.Context, cmd ActivateVenue) error {
	id, err := domain.ParseVenueID(cmd.VenueID)
	if err != nil {
		return fmt.Errorf("activate venue: %w", err)
	}
	venue, err := h.venues.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("activate venue: %w", err)
	}
	if err := venue.Activate(h.clock.Now()); err != nil {
		return fmt.Errorf("activate venue: %w", err)
	}
	if err := h.venues.Save(ctx, venue); err != nil {
		return fmt.Errorf("activate venue: %w", err)
	}
	return nil
}
