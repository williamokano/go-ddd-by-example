package memory

import (
	"context"
	"fmt"
	"slices"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// SeatQueries implements application.SeatQueries over an InventoryRepository.
type SeatQueries struct{ repo *InventoryRepository }

// NewSeatQueries reads from repo.
func NewSeatQueries(repo *InventoryRepository) SeatQueries { return SeatQueries{repo: repo} }

// ListSeats implements application.SeatQueries.
func (q SeatQueries) ListSeats(ctx context.Context, showID domain.ShowID) ([]application.SeatRow, error) {
	sections, err := q.repo.ListByShow(ctx, showID)
	if err != nil {
		return nil, err
	}
	if len(sections) == 0 {
		return nil, fmt.Errorf("%w: show %s", application.ErrInventoryNotFound, showID)
	}
	slices.SortFunc(sections, func(a, b *domain.SectionInventory) int { return a.Position() - b.Position() })
	var rows []application.SeatRow
	for _, inv := range sections {
		for _, s := range inv.Seats() {
			rows = append(rows, application.SeatRow{
				Ref: s.Ref.String(), State: s.State.String(), Amount: s.Price.Amount(), Currency: s.Price.Currency().String(),
			})
		}
	}
	return rows, nil
}
