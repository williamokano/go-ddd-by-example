package application

import (
	"context"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// InventoryRepository loads and saves ShowInventory aggregates.
type InventoryRepository interface {
	// Get loads a show's inventory, or returns ErrInventoryNotFound.
	Get(ctx context.Context, showID domain.ShowID) (*domain.ShowInventory, error)

	// GetByHold loads the inventory holding an active hold, or returns
	// ErrHoldNotFound.
	GetByHold(ctx context.Context, holdID domain.HoldID) (*domain.ShowInventory, error)

	// Save persists the inventory and its events atomically. Opening an
	// inventory that already exists, or saving a stale version, is
	// ErrConcurrentModification.
	Save(ctx context.Context, inv *domain.ShowInventory) error
}

// Clock tells the time (ADR-008).
type Clock interface {
	Now() time.Time
}

// IDGenerator hands out Ticketing's identities.
type IDGenerator interface {
	NewHoldID() domain.HoldID
	NewOrderID() domain.OrderID
}
