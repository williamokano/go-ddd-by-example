package memory

import (
	"context"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// ExpiredHolds implements application.ExpiredHolds over an InventoryRepository.
type ExpiredHolds struct{ repo *InventoryRepository }

// NewExpiredHolds reads from repo.
func NewExpiredHolds(repo *InventoryRepository) ExpiredHolds { return ExpiredHolds{repo: repo} }

// ShowsWithExpiredHolds implements application.ExpiredHolds.
func (e ExpiredHolds) ShowsWithExpiredHolds(_ context.Context, now time.Time) ([]domain.ShowID, error) {
	e.repo.mu.Lock()
	defer e.repo.mu.Unlock()
	var out []domain.ShowID
	for id, state := range e.repo.inventories {
		for _, h := range state.Holds {
			if !now.Before(h.ExpiresAt) {
				out = append(out, id)
				break
			}
		}
	}
	return out, nil
}
