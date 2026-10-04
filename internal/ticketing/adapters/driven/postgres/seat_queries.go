package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	pgplatform "github.com/williamokano/go-ddd-by-example/internal/platform/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres/sqlcgen"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// SeatQueries implements application.SeatQueries with plain SQL.
type SeatQueries struct{ pool *pgxpool.Pool }

// NewSeatQueries returns the queries using pool.
func NewSeatQueries(pool *pgxpool.Pool) *SeatQueries { return &SeatQueries{pool: pool} }

// ListSeats implements application.SeatQueries.
func (q *SeatQueries) ListSeats(ctx context.Context, showID domain.ShowID) ([]application.SeatRow, error) {
	db := sqlcgen.New(pgplatform.Conn(ctx, q.pool))
	exists, err := db.InventoryExists(ctx, showID.UUID())
	if err != nil {
		return nil, fmt.Errorf("list seats %s: %w", showID, err)
	}
	if !exists {
		return nil, fmt.Errorf("%w: show %s", application.ErrInventoryNotFound, showID)
	}
	seats, err := db.ListShowSeats(ctx, showID.UUID())
	if err != nil {
		return nil, fmt.Errorf("list seats %s: %w", showID, err)
	}
	rows := make([]application.SeatRow, len(seats))
	for i, s := range seats {
		rows[i] = application.SeatRow{Ref: s.SeatRef, State: s.State, Amount: s.PriceAmount, Currency: s.Currency}
	}
	return rows, nil
}
