package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// The persistence model (two tables, rows as JSONB) differs from the domain
// model (aggregate → entities → value objects). Translating between them is
// this adapter's job, and only its job.

const (
	kindSeated = "seated"
	kindGA     = "ga"
)

var statuses = map[string]domain.Status{
	domain.Draft.String():   domain.Draft,
	domain.Active.String():  domain.Active,
	domain.Retired.String(): domain.Retired,
}

type rowJSON struct {
	Label      string `json:"label"`
	Seats      int    `json:"seats"`
	Accessible []int  `json:"accessible,omitempty"`
}

// rowsToVenue rebuilds the aggregate from its rows (reconstitution: no
// events, no rules re-run beyond the value objects' own parsing).
func rowsToVenue(v sqlcgen.GetVenueRow, sections []sqlcgen.ListSectionsRow) (*domain.Venue, error) {
	addr, err := domain.NewAddress(v.Street, v.City, v.Country)
	if err != nil {
		return nil, fmt.Errorf("venue %s: stored address: %w", v.ID, err)
	}
	status, ok := statuses[v.Status]
	if !ok {
		return nil, fmt.Errorf("venue %s: unknown stored status %q", v.ID, v.Status)
	}
	state := domain.VenueState{
		ID:      domain.NewVenueID(v.ID),
		Name:    v.Name,
		Address: addr,
		Status:  status,
		Version: int(v.Version),
	}
	for _, s := range sections {
		section, err := rowToSection(s)
		if err != nil {
			return nil, fmt.Errorf("venue %s: %w", v.ID, err)
		}
		state.Sections = append(state.Sections, section)
	}
	return domain.RehydrateVenue(state), nil
}

func rowToSection(s sqlcgen.ListSectionsRow) (domain.Section, error) {
	code, err := domain.NewSectionCode(s.Code)
	if err != nil {
		return domain.Section{}, fmt.Errorf("stored section code: %w", err)
	}
	switch s.Kind {
	case kindGA:
		return domain.NewGeneralAdmissionSection(code, s.Name, int(s.Capacity))
	case kindSeated:
		var stored []rowJSON
		if err := json.Unmarshal(s.Rows, &stored); err != nil {
			return domain.Section{}, fmt.Errorf("section %s: stored rows: %w", s.Code, err)
		}
		rows := make([]domain.Row, 0, len(stored))
		for _, r := range stored {
			row, err := domain.NewRow(r.Label, r.Seats)
			if err == nil {
				row, err = row.WithAccessibleSeats(r.Accessible...)
			}
			if err != nil {
				return domain.Section{}, fmt.Errorf("section %s: stored row: %w", s.Code, err)
			}
			rows = append(rows, row)
		}
		return domain.NewSeatedSection(code, s.Name, rows)
	default:
		return domain.Section{}, fmt.Errorf("section %s: unknown stored kind %q", s.Code, s.Kind)
	}
}

func kindOf(s domain.Section) string {
	if s.Kind() == domain.GeneralAdmission {
		return kindGA
	}
	return kindSeated
}

func rowsJSON(s domain.Section) ([]byte, error) {
	rows := make([]rowJSON, 0, len(s.Rows()))
	for _, r := range s.Rows() {
		rows = append(rows, rowJSON{Label: r.Label(), Seats: r.Seats(), Accessible: r.AccessibleSeats()})
	}
	b, err := json.Marshal(rows)
	if err != nil {
		return nil, fmt.Errorf("section %s: rows: %w", s.Code(), err)
	}
	return b, nil
}
