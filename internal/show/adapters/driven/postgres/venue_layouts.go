// Package postgres implements Show's driven ports on Postgres (schema show).
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pgplatform "github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// VenueLayouts implements application.VenueLayouts on show.venue_layouts.
type VenueLayouts struct{ pool *pgxpool.Pool }

// NewVenueLayouts returns the projection store using pool.
func NewVenueLayouts(pool *pgxpool.Pool) *VenueLayouts { return &VenueLayouts{pool: pool} }

type sectionJSON struct {
	Code     string    `json:"code"`
	Kind     string    `json:"kind"`
	Rows     []rowJSON `json:"rows,omitempty"`
	Capacity int       `json:"capacity,omitempty"`
}

type rowJSON struct {
	Label string `json:"label"`
	Seats int    `json:"seats"`
}

// Get implements application.VenueLayouts.
func (l *VenueLayouts) Get(ctx context.Context, id domain.VenueID) (domain.VenueLayout, error) {
	row, err := sqlcgen.New(pgplatform.Conn(ctx, l.pool)).GetVenueLayout(ctx, uuid.MustParse(id.String()))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.VenueLayout{}, fmt.Errorf("%w: %s", application.ErrVenueUnknown, id)
	}
	if err != nil {
		return domain.VenueLayout{}, fmt.Errorf("get venue layout %s: %w", id, err)
	}
	var sections []sectionJSON
	if err := json.Unmarshal(row.Sections, &sections); err != nil {
		return domain.VenueLayout{}, fmt.Errorf("venue layout %s sections: %w", id, err)
	}
	layout := domain.VenueLayout{VenueID: id, Name: row.Name, Active: row.Active}
	for _, s := range sections {
		ls := domain.LayoutSection{Code: s.Code, Kind: s.Kind, Capacity: s.Capacity}
		for _, r := range s.Rows {
			ls.Rows = append(ls.Rows, domain.LayoutRow{Label: r.Label, Seats: r.Seats})
		}
		layout.Sections = append(layout.Sections, ls)
	}
	return layout, nil
}

// Upsert implements application.VenueLayouts.
func (l *VenueLayouts) Upsert(ctx context.Context, layout domain.VenueLayout) error {
	sections := make([]sectionJSON, 0, len(layout.Sections))
	for _, s := range layout.Sections {
		sj := sectionJSON{Code: s.Code, Kind: s.Kind, Capacity: s.Capacity}
		for _, r := range s.Rows {
			sj.Rows = append(sj.Rows, rowJSON{Label: r.Label, Seats: r.Seats})
		}
		sections = append(sections, sj)
	}
	b, err := json.Marshal(sections)
	if err != nil {
		return fmt.Errorf("venue layout %s: %w", layout.VenueID, err)
	}
	err = sqlcgen.New(pgplatform.Conn(ctx, l.pool)).UpsertVenueLayout(ctx, sqlcgen.UpsertVenueLayoutParams{
		VenueID: uuid.MustParse(layout.VenueID.String()), Name: layout.Name, Active: layout.Active, Sections: b,
	})
	if err != nil {
		return fmt.Errorf("upsert venue layout %s: %w", layout.VenueID, err)
	}
	return nil
}
