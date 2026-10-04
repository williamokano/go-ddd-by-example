package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	pgplatform "github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// ExpiredHolds implements application.ExpiredHolds with an indexed query.
type ExpiredHolds struct{ pool *pgxpool.Pool }

// NewExpiredHolds returns the query using pool.
func NewExpiredHolds(pool *pgxpool.Pool) *ExpiredHolds { return &ExpiredHolds{pool: pool} }

// SectionsWithExpiredHolds implements application.ExpiredHolds.
func (e *ExpiredHolds) SectionsWithExpiredHolds(ctx context.Context, now time.Time) ([]application.SectionKey, error) {
	rows, err := sqlcgen.New(pgplatform.Conn(ctx, e.pool)).SectionsWithExpiredHolds(ctx, now)
	if err != nil {
		return nil, fmt.Errorf("sections with expired holds: %w", err)
	}
	out := make([]application.SectionKey, len(rows))
	for i, r := range rows {
		out[i] = application.SectionKey{ShowID: domain.NewShowID(r.ShowID), Section: r.Section}
	}
	return out, nil
}
