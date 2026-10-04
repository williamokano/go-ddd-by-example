package application

import (
	"context"
	"errors"
	"fmt"
)

// ExpireHoldsHandler frees the seats of lapsed holds (TKT-4). A scheduler
// drives it; it is a use case like any other.
type ExpireHoldsHandler struct {
	inventories InventoryRepository
	expired     ExpiredHolds
	clock       Clock
}

// NewExpireHoldsHandler wires the use case.
func NewExpireHoldsHandler(inventories InventoryRepository, expired ExpiredHolds, clock Clock) *ExpireHoldsHandler {
	return &ExpireHoldsHandler{inventories: inventories, expired: expired, clock: clock}
}

// Handle sweeps every show with lapsed holds, one inventory transaction each.
func (h *ExpireHoldsHandler) Handle(ctx context.Context) error {
	now := h.clock.Now()
	shows, err := h.expired.ShowsWithExpiredHolds(ctx, now)
	if err != nil {
		return fmt.Errorf("expire holds: %w", err)
	}
	var errs []error
	for _, showID := range shows {
		err := RetryOnConflict(ctx, conflictAttempts, func(ctx context.Context) error {
			inv, err := h.inventories.Get(ctx, showID)
			if err != nil {
				return fmt.Errorf("load: %w", err)
			}
			inv.ExpireHolds(now)
			if err := h.inventories.Save(ctx, inv); err != nil {
				return fmt.Errorf("save: %w", err)
			}
			return nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("show %s: %w", showID, err))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("expire holds: %w", err)
	}
	return nil
}
