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

// ExpiredHolds finds the shows that have holds lapsed at now (TKT-4).
type ExpiredHolds interface {
	ShowsWithExpiredHolds(ctx context.Context, now time.Time) ([]domain.ShowID, error)
}

// Clock tells the time (ADR-008).
type Clock interface {
	Now() time.Time
}

// OrderRepository loads and saves Order aggregates.
type OrderRepository interface {
	// Get loads an order, or returns ErrOrderNotFound.
	Get(ctx context.Context, id domain.OrderID) (*domain.Order, error)

	// Save persists the order and its events atomically (ADR-004, ADR-011).
	Save(ctx context.Context, o *domain.Order) error

	// ListPaidForShow returns the show's Paid and Fulfilled orders (TKT-11).
	ListPaidForShow(ctx context.Context, showID domain.ShowID) ([]*domain.Order, error)
}

// TicketRepository loads and saves Ticket aggregates.
type TicketRepository interface {
	// Save persists a ticket. Saving a new ticket whose ID already exists is
	// a no-op: ticket IDs derive from order + seat, so a redelivered "issue
	// tickets" message can't create duplicates (TKT-9).
	Save(ctx context.Context, t *domain.Ticket) error

	// ListByOrder returns an order's tickets.
	ListByOrder(ctx context.Context, orderID domain.OrderID) ([]*domain.Ticket, error)

	// ListByShow returns a show's tickets.
	ListByShow(ctx context.Context, showID domain.ShowID) ([]*domain.Ticket, error)
}

// IDGenerator hands out Ticketing's identities.
type IDGenerator interface {
	NewHoldID() domain.HoldID
	NewOrderID() domain.OrderID

	// TicketIDFor derives the ticket ID of an order's seat: the same inputs
	// always give the same ID.
	TicketIDFor(order domain.OrderID, seat domain.SeatRef) domain.TicketID
}
