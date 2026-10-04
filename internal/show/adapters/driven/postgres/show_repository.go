package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	pgplatform "github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// ShowRepository implements application.ShowRepository on show.shows, writing
// integration events to show.outbox in the same transaction (ADR-004).
type ShowRepository struct {
	pool       *pgxpool.Pool
	newEventID func() uuid.UUID
}

// NewShowRepository returns a repository using pool.
func NewShowRepository(pool *pgxpool.Pool) *ShowRepository {
	return &ShowRepository{pool: pool, newEventID: idgen.UUIDv7{}.New}
}

// Get implements application.ShowRepository.
func (r *ShowRepository) Get(ctx context.Context, id domain.ShowID) (*domain.Show, error) {
	row, err := sqlcgen.New(pgplatform.Conn(ctx, r.pool)).GetShow(ctx, uuid.MustParse(id.String()))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", application.ErrShowNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get show %s: %w", id, err)
	}
	return rowToShow(showRow(row))
}

// ListOpenAtVenue implements application.ShowRepository.
func (r *ShowRepository) ListOpenAtVenue(ctx context.Context, venueID domain.VenueID) ([]*domain.Show, error) {
	rows, err := sqlcgen.New(pgplatform.Conn(ctx, r.pool)).ListOpenShowsAtVenue(ctx, uuid.MustParse(venueID.String()))
	if err != nil {
		return nil, fmt.Errorf("list shows at venue %s: %w", venueID, err)
	}
	shows := make([]*domain.Show, 0, len(rows))
	for _, row := range rows {
		s, err := rowToShow(showRow(row))
		if err != nil {
			return nil, err
		}
		shows = append(shows, s)
	}
	return shows, nil
}

// ListEnded implements application.ShowRepository.
func (r *ShowRepository) ListEnded(ctx context.Context, now time.Time) ([]domain.ShowID, error) {
	ids, err := sqlcgen.New(pgplatform.Conn(ctx, r.pool)).ListEndedShows(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("list ended shows: %w", err)
	}
	out := make([]domain.ShowID, len(ids))
	for i, id := range ids {
		out[i] = domain.NewShowID(id)
	}
	return out, nil
}

// Save implements application.ShowRepository.
func (r *ShowRepository) Save(ctx context.Context, s *domain.Show) (err error) {
	prices, err := pricesJSON(s.Prices())
	if err != nil {
		return err
	}
	tx, err := pgplatform.Begin(ctx, r.pool)
	if err != nil {
		return fmt.Errorf("save show %s: begin: %w", s.ID(), err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	q := sqlcgen.New(tx)
	id := uuid.MustParse(s.ID().String())
	sched := s.Schedule()
	var n int64
	if s.Version() == 0 {
		n, err = q.InsertShow(ctx, sqlcgen.InsertShowParams{
			ID: id, VenueID: uuid.MustParse(s.VenueID().String()), PromoterID: uuid.MustParse(s.PromoterID().String()),
			Title: s.Title(), DoorsOpen: sched.DoorsOpen(), StartsAt: sched.StartsAt(), EndsAt: sched.EndsAt(),
			Status: s.Status().String(), Prices: prices, CancellationReason: s.CancellationReason().String(),
		})
	} else {
		n, err = q.UpdateShow(ctx, sqlcgen.UpdateShowParams{
			ID: id, Title: s.Title(), DoorsOpen: sched.DoorsOpen(), StartsAt: sched.StartsAt(), EndsAt: sched.EndsAt(),
			Status: s.Status().String(), Prices: prices, CancellationReason: s.CancellationReason().String(),
			ExpectedVersion: int32(s.Version()),
		})
	}
	if err != nil {
		return fmt.Errorf("save show %s: %w", s.ID(), err)
	}
	if n == 0 {
		return fmt.Errorf("%w: show %s", application.ErrConcurrentModification, s.ID())
	}
	msgs, err := ToOutboxMessages(s.PullEvents(), r.newEventID)
	if err != nil {
		return fmt.Errorf("save show %s: %w", s.ID(), err)
	}
	if err := outbox.Write(ctx, tx, "show", msgs); err != nil {
		return fmt.Errorf("save show %s: %w", s.ID(), err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("save show %s: commit: %w", s.ID(), err)
	}
	return nil
}
