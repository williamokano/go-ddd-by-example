package application

import (
	"context"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// AddSection is the command to add a section to a draft venue's layout.
// Kind is "seated" (with Rows) or "ga" (with Capacity).
type AddSection struct {
	VenueID  string
	Code     string
	Name     string
	Kind     string
	Rows     []RowSpec
	Capacity int
}

// RowSpec describes one row of a seated section.
type RowSpec struct {
	Label string
	Seats int
}

// AddSectionHandler is the AddSection use case.
type AddSectionHandler struct {
	venues VenueRepository
	clock  Clock
}

// NewAddSectionHandler wires the use case to its driven ports.
func NewAddSectionHandler(venues VenueRepository, clock Clock) *AddSectionHandler {
	return &AddSectionHandler{venues: venues, clock: clock}
}

// Handle adds the section.
func (h *AddSectionHandler) Handle(ctx context.Context, cmd AddSection) error {
	id, _ := domain.ParseVenueID(cmd.VenueID)
	code, _ := domain.NewSectionCode(cmd.Code)
	rows := make([]domain.Row, 0, len(cmd.Rows))
	for _, r := range cmd.Rows {
		row, _ := domain.NewRow(r.Label, r.Seats)
		rows = append(rows, row)
	}
	section, _ := domain.NewSeatedSection(code, cmd.Name, rows)
	venue, _ := h.venues.Get(ctx, id)
	_ = venue.AddSection(section, h.clock.Now())
	if err := h.venues.Save(ctx, venue); err != nil {
		return fmt.Errorf("add section: %w", err)
	}
	return nil
}
