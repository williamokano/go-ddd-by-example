package application

import (
	"context"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
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

// PaymentGateway charges and refunds money through an external provider.
// It speaks our language (Charge, Refund), not the provider's (intents,
// captures, webhooks): the port is an anti-corruption layer, and a real
// adapter does the translation.
type PaymentGateway interface {
	// Charge takes amount for the order. The order ID is the idempotency
	// key: charging the same order twice charges once. A refused card is
	// ErrPaymentDeclined.
	Charge(ctx context.Context, orderID domain.OrderID, amount sharedkernel.Money) (domain.PaymentRef, error)

	// Refund gives amount back for a charge.
	Refund(ctx context.Context, ref domain.PaymentRef, amount sharedkernel.Money) error
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
