package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	pgplatform "github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// ExpiredHolds implements application.ExpiredHolds with an indexed query.
type ExpiredHolds struct{ pool *pgxpool.Pool }

// NewExpiredHolds returns the query using pool.
func NewExpiredHolds(pool *pgxpool.Pool) *ExpiredHolds { return &ExpiredHolds{pool: pool} }

// ShowsWithExpiredHolds implements application.ExpiredHolds.
func (e *ExpiredHolds) ShowsWithExpiredHolds(ctx context.Context, now time.Time) ([]domain.ShowID, error) {
	ids, err := sqlcgen.New(pgplatform.Conn(ctx, e.pool)).ShowsWithExpiredHolds(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("shows with expired holds: %w", err)
	}
	out := make([]domain.ShowID, len(ids))
	for i, id := range ids {
		out[i] = domain.NewShowID(id)
	}
	return out, nil
}
