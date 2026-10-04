package memory

import (
	"context"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
)

// ExpiredHolds implements application.ExpiredHolds over an InventoryRepository.
type ExpiredHolds struct{ repo *InventoryRepository }

// NewExpiredHolds reads from repo.
func NewExpiredHolds(repo *InventoryRepository) ExpiredHolds { return ExpiredHolds{repo: repo} }

// SectionsWithExpiredHolds implements application.ExpiredHolds.
func (e ExpiredHolds) SectionsWithExpiredHolds(_ context.Context, now time.Time) ([]application.SectionKey, error) {
	e.repo.mu.Lock()
	defer e.repo.mu.Unlock()
	var out []application.SectionKey
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
