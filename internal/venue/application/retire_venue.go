package application

import (
	"context"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// RetireVenue is the command to close an active venue for good.
type RetireVenue struct{ VenueID string }

// RetireVenueHandler is the RetireVenue use case.
type RetireVenueHandler struct {
	venues VenueRepository
	clock  Clock
}

// NewRetireVenueHandler wires the use case to its driven ports.
func NewRetireVenueHandler(venues VenueRepository, clock Clock) *RetireVenueHandler {
	return &RetireVenueHandler{venues: venues, clock: clock}
}

// Handle retires the venue.
func (h *RetireVenueHandler) Handle(ctx context.Context, cmd RetireVenue) error {
	id, err := domain.ParseVenueID(cmd.VenueID)
	if err != nil {
		return fmt.Errorf("retire venue: %w", err)
	}
	venue, err := h.venues.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("retire venue: %w", err)
	}
	if err := venue.Retire(h.clock.Now()); err != nil {
		return fmt.Errorf("retire venue: %w", err)
	}
	if err := h.venues.Save(ctx, venue); err != nil {
		return fmt.Errorf("retire venue: %w", err)
	}
	return nil
}
