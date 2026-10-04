package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// OnVenueActivated is Show's own command for "a venue opened": the consumer
// translates venue.activated.v1 into it (the anti-corruption layer), so
// Venue's contract never reaches past the adapter.
type OnVenueActivated struct {
	VenueID  string
	Name     string
	Sections []LayoutSectionSpec
}

// LayoutSectionSpec is one section of an activated venue.
type LayoutSectionSpec struct {
	Code     string
	Kind     string
	Rows     []RowSpec
	Capacity int
}

// RowSpec is one row of a seated section.
type RowSpec struct {
	Label string
	Seats int
}

// OnVenueActivatedHandler keeps the local VenueLayout projection up to date.
type OnVenueActivatedHandler struct{ layouts VenueLayouts }

// NewOnVenueActivatedHandler wires the policy to its port.
func NewOnVenueActivatedHandler(layouts VenueLayouts) *OnVenueActivatedHandler {
	return &OnVenueActivatedHandler{layouts: layouts}
}

// Handle stores the venue's layout. An upsert: receiving the fact twice
// leaves the same state (at-least-once delivery).
func (h *OnVenueActivatedHandler) Handle(ctx context.Context, cmd OnVenueActivated) error {
	id, err := domain.ParseVenueID(cmd.VenueID)
	if err != nil {
		return fmt.Errorf("on venue activated: %w", err)
	}
	layout := domain.VenueLayout{VenueID: id, Name: cmd.Name, Active: true}
	for _, s := range cmd.Sections {
		section := domain.LayoutSection{Code: s.Code, Kind: s.Kind, Capacity: s.Capacity}
		for _, r := range s.Rows {
			section.Rows = append(section.Rows, domain.LayoutRow{Label: r.Label, Seats: r.Seats})
		}
		layout.Sections = append(layout.Sections, section)
	}
	if err := h.layouts.Upsert(ctx, layout); err != nil {
		return fmt.Errorf("on venue activated: %w", err)
	}
	return nil
}

// OnVenueRetired is Show's command for "a venue closed for good".
type OnVenueRetired struct{ VenueID string }

// OnVenueRetiredHandler is the SHW-7 policy: whenever a venue retires, cancel
// each of its future shows.
type OnVenueRetiredHandler struct {
	layouts VenueLayouts
	shows   ShowRepository
	clock   Clock
}

// NewOnVenueRetiredHandler wires the policy to its ports.
func NewOnVenueRetiredHandler(layouts VenueLayouts, shows ShowRepository, clock Clock) *OnVenueRetiredHandler {
	return &OnVenueRetiredHandler{layouts: layouts, shows: shows, clock: clock}
}

// Handle marks the venue inactive in the projection, then cancels every open
// show there that hasn't started yet, one aggregate (one transaction) per
// show. Re-running it is harmless: cancelled shows are no longer open.
func (h *OnVenueRetiredHandler) Handle(ctx context.Context, cmd OnVenueRetired) error {
	id, err := domain.ParseVenueID(cmd.VenueID)
	if err != nil {
		return fmt.Errorf("on venue retired: %w", err)
	}
	layout, err := h.layouts.Get(ctx, id)
	if errors.Is(err, ErrVenueUnknown) {
		layout = domain.VenueLayout{VenueID: id}
	} else if err != nil {
		return fmt.Errorf("on venue retired: %w", err)
	}
	layout.Active = false
	if err := h.layouts.Upsert(ctx, layout); err != nil {
		return fmt.Errorf("on venue retired: %w", err)
	}

	shows, err := h.shows.ListOpenAtVenue(ctx, id)
	if err != nil {
		return fmt.Errorf("on venue retired: %w", err)
	}
	reason, err := domain.NewCancellationReason(domain.ReasonVenueRetired)
	if err != nil {
		return err
	}
	for _, s := range shows {
		if !s.Schedule().StartsAt().After(h.clock.Now()) {
			continue
		}
		err := RetryOnConflict(ctx, conflictAttempts, func(ctx context.Context) error {
			show, err := h.shows.Get(ctx, s.ID())
			if err != nil {
				return fmt.Errorf("load: %w", err)
			}
			if show.Status().IsTerminal() {
				return nil
			}
			if err := show.Cancel(reason, h.clock.Now()); err != nil {
				return err
			}
			if err := h.shows.Save(ctx, show); err != nil {
				return fmt.Errorf("save: %w", err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("on venue retired: cancel show %s: %w", s.ID(), err)
		}
	}
	return nil
}
