package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// VenueQueries implements application.VenueQueries with plain SQL: no
// aggregate is loaded, and capacity is summed by Postgres.
type VenueQueries struct{ pool *pgxpool.Pool }

// NewVenueQueries returns the queries using pool.
func NewVenueQueries(pool *pgxpool.Pool) *VenueQueries { return &VenueQueries{pool: pool} }

// Get implements application.VenueQueries.
func (q *VenueQueries) Get(ctx context.Context, id domain.VenueID) (application.VenueView, error) {
	uid, err := uuid.Parse(id.String())
	if err != nil {
		return application.VenueView{}, fmt.Errorf("venue id: %w", err)
	}
	db := sqlcgen.New(q.pool)
	row, err := db.GetVenueView(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return application.VenueView{}, fmt.Errorf("%w: %s", application.ErrVenueNotFound, id)
	}
	if err != nil {
		return application.VenueView{}, fmt.Errorf("get venue view %s: %w", id, err)
	}
	sections, err := db.ListSections(ctx, uid)
	if err != nil {
		return application.VenueView{}, fmt.Errorf("get venue view %s sections: %w", id, err)
	}
	view := application.VenueView{
		ID: row.ID.String(), Name: row.Name, Street: row.Street, City: row.City,
		Country: row.Country, Status: row.Status, Capacity: int(row.Capacity),
	}
	for _, s := range sections {
		sv, err := sectionView(s.Code, s.Name, s.Kind, s.Capacity, s.Rows)
		if err != nil {
			return application.VenueView{}, err
		}
		view.Sections = append(view.Sections, sv)
	}
	return view, nil
}

// List implements application.VenueQueries.
func (q *VenueQueries) List(ctx context.Context, status string) ([]application.VenueView, error) {
	db := sqlcgen.New(q.pool)
	rows, err := db.ListVenueViewsByStatus(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("list venues %s: %w", status, err)
	}
	ids := make([]uuid.UUID, len(rows))
	views := make([]application.VenueView, len(rows))
	index := make(map[uuid.UUID]int, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
		index[r.ID] = i
		views[i] = application.VenueView{
			ID: r.ID.String(), Name: r.Name, Street: r.Street, City: r.City,
			Country: r.Country, Status: r.Status, Capacity: int(r.Capacity),
		}
	}
	sections, err := db.ListSectionsOfVenues(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("list venues %s sections: %w", status, err)
	}
	for _, s := range sections {
		sv, err := sectionView(s.Code, s.Name, s.Kind, s.Capacity, s.Rows)
		if err != nil {
			return nil, err
		}
		i := index[s.VenueID]
		views[i].Sections = append(views[i].Sections, sv)
	}
	return views, nil
}

func sectionView(code, name, kind string, capacity int32, rowsJSONB []byte) (application.SectionView, error) {
	sv := application.SectionView{Code: code, Name: name, Kind: kind, Capacity: int(capacity)}
	var rows []rowJSON
	if err := json.Unmarshal(rowsJSONB, &rows); err != nil {
		return application.SectionView{}, fmt.Errorf("section %s rows: %w", code, err)
	}
	for _, r := range rows {
		sv.Rows = append(sv.Rows, application.RowView{Label: r.Label, Seats: r.Seats})
	}
	return sv, nil
}
