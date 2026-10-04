package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// ShowSchedule implements application.ShowSchedule over an InventoryRepository.
type ShowSchedule struct{ repo *InventoryRepository }

// NewShowSchedule reads from repo.
func NewShowSchedule(repo *InventoryRepository) ShowSchedule { return ShowSchedule{repo: repo} }

// StartsAt implements application.ShowSchedule.
func (s ShowSchedule) StartsAt(_ context.Context, showID domain.ShowID) (time.Time, error) {
	s.repo.mu.Lock()
	defer s.repo.mu.Unlock()
	for key, state := range s.repo.inventories {
		if key.ShowID == showID {
			return state.StartsAt, nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: show %s", application.ErrInventoryNotFound, showID)
}
