package memory

import (
	"context"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// SeatQueries implements application.SeatQueries over an InventoryRepository.
type SeatQueries struct{ repo *InventoryRepository }

// NewSeatQueries reads from repo.
func NewSeatQueries(repo *InventoryRepository) SeatQueries { return SeatQueries{repo: repo} }

// ListSeats implements application.SeatQueries.
func (q SeatQueries) ListSeats(ctx context.Context, showID domain.ShowID) ([]application.SeatRow, error) {
	inv, err := q.repo.Get(ctx, showID)
	if err != nil {
		return nil, err
	}
	rows := make([]application.SeatRow, 0, len(inv.Seats()))
	for _, s := range inv.Seats() {
		rows = append(rows, application.SeatRow{
			Ref: s.Ref.String(), State: s.State.String(), Amount: s.Price.Amount(), Currency: s.Price.Currency().String(),
		})
	}
	return rows, nil
}
