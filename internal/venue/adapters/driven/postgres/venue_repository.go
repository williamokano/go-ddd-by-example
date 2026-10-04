// Package postgres implements the Venue context's driven ports on Postgres:
// the VenueRepository (aggregate in, aggregate out) and VenueQueries (flat
// views straight from SQL). sqlc-generated types never leave this package.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	pgplatform "github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// VenueRepository implements application.VenueRepository on Postgres.
type VenueRepository struct {
	pool       *pgxpool.Pool
	newEventID func() uuid.UUID
}

// NewVenueRepository returns a repository using pool.
func NewVenueRepository(pool *pgxpool.Pool) *VenueRepository {
	return &VenueRepository{pool: pool, newEventID: idgen.UUIDv7{}.New}
}

// Get implements application.VenueRepository.
func (r *VenueRepository) Get(ctx context.Context, id domain.VenueID) (*domain.Venue, error) {
	uid, err := uuid.Parse(id.String())
	if err != nil {
		return nil, fmt.Errorf("venue id: %w", err)
	}
	q := sqlcgen.New(pgplatform.Conn(ctx, r.pool))
	row, err := q.GetVenue(ctx, uid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", application.ErrVenueNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get venue %s: %w", id, err)
	}
	sections, err := q.ListSections(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("get venue %s sections: %w", id, err)
	}
	return rowsToVenue(row, sections)
}

// Save implements application.VenueRepository: one transaction writes the
// venue row (insert, or update guarded by the version), replaces its sections
// (fine: layouts only change while the venue is a draft) and writes the
// venue's integration events to the outbox (ADR-004).
func (r *VenueRepository) Save(ctx context.Context, v *domain.Venue) (err error) {
	uid, err := uuid.Parse(v.ID().String())
	if err != nil {
		return fmt.Errorf("venue id: %w", err)
	}
	tx, err := pgplatform.Begin(ctx, r.pool)
	if err != nil {
		return fmt.Errorf("save venue %s: begin: %w", v.ID(), err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	q := sqlcgen.New(tx)

	if err := saveVenueRow(ctx, q, uid, v); err != nil {
		return err
	}
	if err := q.DeleteSections(ctx, uid); err != nil {
		return fmt.Errorf("save venue %s: delete sections: %w", v.ID(), err)
	}
	for i, s := range v.Sections() {
		rows, err := rowsJSON(s)
		if err != nil {
			return err
		}
		if err := q.InsertSection(ctx, sqlcgen.InsertSectionParams{
			VenueID: uid, Code: s.Code().String(), Position: int32(i), Name: s.Name(),
			Kind: kindOf(s), Capacity: int32(s.Capacity()), Rows: rows,
		}); err != nil {
			return fmt.Errorf("save venue %s: insert section %s: %w", v.ID(), s.Code(), err)
		}
	}
	msgs, err := ToOutboxMessages(v.PullEvents(), r.newEventID)
	if err != nil {
		return fmt.Errorf("save venue %s: %w", v.ID(), err)
	}
	if err := outbox.Write(ctx, tx, "venue", msgs); err != nil {
		return fmt.Errorf("save venue %s: %w", v.ID(), err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("save venue %s: commit: %w", v.ID(), err)
	}
	return nil
}

// saveVenueRow inserts a new venue (version 0 in memory) or updates a loaded
// one; 0 affected rows means someone else saved first (ADR-011).
func saveVenueRow(ctx context.Context, q *sqlcgen.Queries, uid uuid.UUID, v *domain.Venue) error {
	var (
		n   int64
		err error
	)
	addr := v.Address()
	if v.Version() == 0 {
		n, err = q.InsertVenue(ctx, sqlcgen.InsertVenueParams{
			ID: uid, Name: v.Name(), Street: addr.Street(), City: addr.City(), Country: addr.Country(), Status: v.Status().String(),
		})
	} else {
		n, err = q.UpdateVenue(ctx, sqlcgen.UpdateVenueParams{
			ID: uid, Name: v.Name(), Street: addr.Street(), City: addr.City(), Country: addr.Country(), Status: v.Status().String(),
			ExpectedVersion: int32(v.Version()),
		})
	}
	if err != nil {
		return fmt.Errorf("save venue %s: %w", v.ID(), err)
	}
	if n == 0 {
		return fmt.Errorf("%w: venue %s", application.ErrConcurrentModification, v.ID())
	}
	return nil
}
