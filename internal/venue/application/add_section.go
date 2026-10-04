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

// Section kinds as they appear in commands.
const (
	KindSeated = "seated"
	KindGA     = "ga"
)

// AddSectionHandler is the AddSection use case.
type AddSectionHandler struct {
	venues VenueRepository
	clock  Clock
}

// NewAddSectionHandler wires the use case to its driven ports.
func NewAddSectionHandler(venues VenueRepository, clock Clock) *AddSectionHandler {
	return &AddSectionHandler{venues: venues, clock: clock}
}

// Handle adds the section: parse → load → one domain call → save.
func (h *AddSectionHandler) Handle(ctx context.Context, cmd AddSection) error {
	id, err := domain.ParseVenueID(cmd.VenueID)
	if err != nil {
		return fmt.Errorf("add section: %w", err)
	}
	section, err := newSection(cmd)
	if err != nil {
		return fmt.Errorf("add section: %w", err)
	}
	venue, err := h.venues.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("add section: %w", err)
	}
	if err := venue.AddSection(section, h.clock.Now()); err != nil {
		return fmt.Errorf("add section: %w", err)
	}
	if err := h.venues.Save(ctx, venue); err != nil {
		return fmt.Errorf("add section: %w", err)
	}
	return nil
}

// newSection parses the command's primitives into a domain Section.
func newSection(cmd AddSection) (domain.Section, error) {
	code, err := domain.NewSectionCode(cmd.Code)
	if err != nil {
		return domain.Section{}, err
	}
	switch cmd.Kind {
	case KindSeated:
		rows := make([]domain.Row, 0, len(cmd.Rows))
		for _, r := range cmd.Rows {
			row, err := domain.NewRow(r.Label, r.Seats)
			if err != nil {
				return domain.Section{}, err
			}
			rows = append(rows, row)
		}
		return domain.NewSeatedSection(code, cmd.Name, rows)
	case KindGA:
		return domain.NewGeneralAdmissionSection(code, cmd.Name, cmd.Capacity)
	default:
		return domain.Section{}, fmt.Errorf("%w: unknown kind %q (want %q or %q)", domain.ErrInvalidSection, cmd.Kind, KindSeated, KindGA)
	}
}
