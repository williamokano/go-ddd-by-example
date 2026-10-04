package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// Checkout is the command to buy the seats of a hold. Everything comes from
// the customer: there is no login (Chapter 1).
type Checkout struct {
	HoldID       string
	CustomerID   string
	ContactEmail string
}

// CheckoutResult says which order was placed and how its payment went.
type CheckoutResult struct {
	OrderID domain.OrderID
	Status  domain.OrderStatus
}

// CheckoutHandler places the order and charges it: the first two steps of
// the checkout saga (ADR-010), each its own transaction on the Order.
type CheckoutHandler struct {
	inventories InventoryRepository
	orders      OrderRepository
	gateway     PaymentGateway
	ids         IDGenerator
	clock       Clock
}

// NewCheckoutHandler wires the use case to its ports.
func NewCheckoutHandler(inventories InventoryRepository, orders OrderRepository, gateway PaymentGateway, ids IDGenerator, clock Clock) *CheckoutHandler {
	return &CheckoutHandler{inventories: inventories, orders: orders, gateway: gateway, ids: ids, clock: clock}
}

// Handle places the order (tx1), charges it outside any transaction, and
// records the outcome (tx2). It only reads the inventory: confirming the hold
// is the saga's next step, driven by OrderPaid. A declined card leaves the
// hold in place, so the customer can try again.
func (h *CheckoutHandler) Handle(ctx context.Context, cmd Checkout) (CheckoutResult, error) {
	email, err := domain.NewContactEmail(cmd.ContactEmail)
	if err != nil {
		return CheckoutResult{}, fmt.Errorf("checkout: %w", err)
	}
	customer, err := domain.ParseCustomerID(cmd.CustomerID)
	if err != nil {
		return CheckoutResult{}, fmt.Errorf("checkout: %w", err)
	}
	holdID, err := domain.ParseHoldID(cmd.HoldID)
	if err != nil {
		return CheckoutResult{}, fmt.Errorf("checkout: %w", err)
	}
	inv, err := h.inventories.GetByHold(ctx, holdID)
	if err != nil {
		return CheckoutResult{}, fmt.Errorf("checkout: %w", err)
	}
	hold, err := inv.HoldView(holdID)
	if err != nil {
		return CheckoutResult{}, fmt.Errorf("checkout: %w", err)
	}
	order, err := domain.PlaceOrder(h.ids.NewOrderID(), customer, email, hold, h.clock.Now())
	if err != nil {
		return CheckoutResult{}, fmt.Errorf("checkout: %w", err)
	}
	if err := h.orders.Save(ctx, order); err != nil {
		return CheckoutResult{}, fmt.Errorf("checkout: place: %w", err)
	}

	ref, chargeErr := h.gateway.Charge(ctx, order.ID(), order.Total())
	order, err = h.orders.Get(ctx, order.ID())
	if err != nil {
		return CheckoutResult{}, fmt.Errorf("checkout: reload: %w", err)
	}
	switch {
	case chargeErr == nil:
		err = order.MarkPaid(ref, h.clock.Now())
	case errors.Is(chargeErr, ErrPaymentDeclined):
		err = order.MarkPaymentFailed(chargeErr.Error(), h.clock.Now())
	default:
		// Unknown outcome: the order stays Pending. A reconciliation job
		// (the provider's records, keyed by order ID) settles it.
		return CheckoutResult{}, fmt.Errorf("checkout: charge: %w", chargeErr)
	}
	if err != nil {
		return CheckoutResult{}, fmt.Errorf("checkout: %w", err)
	}
	if err := h.orders.Save(ctx, order); err != nil {
		return CheckoutResult{}, fmt.Errorf("checkout: record payment: %w", err)
	}
	return CheckoutResult{OrderID: order.ID(), Status: order.Status()}, nil
}
