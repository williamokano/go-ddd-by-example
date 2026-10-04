package application

import (
	"context"
	"fmt"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// DraftShow is the command to draft a show at a venue, for a promoter.
type DraftShow struct {
	PromoterID string
	VenueID    string
	Title      string
	DoorsOpen  time.Time
	StartsAt   time.Time
	EndsAt     time.Time
}

// DraftShowHandler is the DraftShow use case.
type DraftShowHandler struct {
	shows   ShowRepository
	layouts VenueLayouts
	ids     IDGenerator
	clock   Clock
	policy  domain.SchedulingPolicy
}

// NewDraftShowHandler wires the use case to its ports.
func NewDraftShowHandler(shows ShowRepository, layouts VenueLayouts, ids IDGenerator, clock Clock) *DraftShowHandler {
	return &DraftShowHandler{shows: shows, layouts: layouts, ids: ids, clock: clock}
}

// Handle drafts the show: parse, load the venue's layout and open shows,
// check the scheduling policy (SHW-3), draft (SHW-1, SHW-2), save.
//
// Two concurrent drafts can both pass the overlap check; a Postgres exclusion
// constraint would be the backstop. The domain rule stays the statement.
func (h *DraftShowHandler) Handle(ctx context.Context, cmd DraftShow) (domain.ShowID, error) {
	promoter, err := domain.ParsePromoterID(cmd.PromoterID)
	if err != nil {
		return domain.ShowID{}, fmt.Errorf("draft show: %w", err)
	}
	venueID, err := domain.ParseVenueID(cmd.VenueID)
	if err != nil {
		return domain.ShowID{}, fmt.Errorf("draft show: %w", err)
	}
	schedule, err := domain.NewSchedule(cmd.DoorsOpen, cmd.StartsAt, cmd.EndsAt)
	if err != nil {
		return domain.ShowID{}, fmt.Errorf("draft show: %w", err)
	}
	layout, err := h.layouts.Get(ctx, venueID)
	if err != nil {
		return domain.ShowID{}, fmt.Errorf("draft show: %w", err)
	}
	existing, err := h.shows.ListOpenAtVenue(ctx, venueID)
	if err != nil {
		return domain.ShowID{}, fmt.Errorf("draft show: %w", err)
	}
	id := h.ids.NewShowID()
	if err := h.policy.EnsureNoOverlap(id, schedule, existing); err != nil {
		return domain.ShowID{}, fmt.Errorf("draft show: %w", err)
	}
	show, err := domain.DraftShow(id, layout, promoter, cmd.Title, schedule, h.clock.Now())
	if err != nil {
		return domain.ShowID{}, fmt.Errorf("draft show: %w", err)
	}
	if err := h.shows.Save(ctx, show); err != nil {
		return domain.ShowID{}, fmt.Errorf("draft show: %w", err)
	}
	return id, nil
}
