package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	pgplatform "github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// ShowSchedule implements application.ShowSchedule with one indexed read.
type ShowSchedule struct{ pool *pgxpool.Pool }

// NewShowSchedule returns the read using pool.
func NewShowSchedule(pool *pgxpool.Pool) *ShowSchedule { return &ShowSchedule{pool: pool} }

// StartsAt implements application.ShowSchedule.
func (s *ShowSchedule) StartsAt(ctx context.Context, showID domain.ShowID) (time.Time, error) {
	at, err := sqlcgen.New(pgplatform.Conn(ctx, s.pool)).ShowStartsAt(ctx, showID.UUID())
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, fmt.Errorf("%w: show %s", application.ErrInventoryNotFound, showID)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("start of show %s: %w", showID, err)
	}
	return at, nil
}
