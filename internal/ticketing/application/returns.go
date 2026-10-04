package application

import (
	"context"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// ReturnOrder is the buyer giving a fulfilled order back (TKT-14).
type ReturnOrder struct {
	OrderID    string
	CustomerID string
}

// ReturnOrderHandler checks the return and refunds the order. The rest is
// the saga's: OrderRefunded puts the seats back on sale and voids the
// tickets, each in its own transaction (9.5).
type ReturnOrderHandler struct {
	orders   OrderRepository
	schedule ShowSchedule
	refund   RefundOrderStep
	clock    Clock
}

// NewReturnOrderHandler wires the use case.
func NewReturnOrderHandler(orders OrderRepository, schedule ShowSchedule, refund RefundOrderStep, clock Clock) *ReturnOrderHandler {
	return &ReturnOrderHandler{orders: orders, schedule: schedule, refund: refund, clock: clock}
}

// Handle returns the order.
func (h *ReturnOrderHandler) Handle(ctx context.Context, cmd ReturnOrder) error {
	orderID, err := domain.ParseOrderID(cmd.OrderID)
	if err != nil {
		return fmt.Errorf("return order: %w", err)
	}
	customer, err := domain.ParseCustomerID(cmd.CustomerID)
	if err != nil {
		return fmt.Errorf("return order: %w", err)
	}
	order, err := h.orders.Get(ctx, orderID)
	if err != nil {
		return fmt.Errorf("return order: %w", err)
	}
	startsAt, err := h.schedule.StartsAt(ctx, order.ShowID())
	if err != nil {
		return fmt.Errorf("return order: %w", err)
	}
	if err := order.CheckReturn(customer, startsAt, h.clock.Now()); err != nil {
		return fmt.Errorf("return order: %w", err)
	}
	if err := h.refund.Handle(ctx, RefundOrder{OrderID: cmd.OrderID}); err != nil {
		return fmt.Errorf("return order: %w", err)
	}
	return nil
}

// OnOrderRefunded is the saga step driven by OrderRefunded.
type OnOrderRefunded struct {
	ShowID  string
	Section string
	OrderID string
}

// OnOrderRefundedHandler voids a refunded order's tickets and puts its seats
// back on sale. Every refund drives it; after a failed confirmation or a
// cancelled show, there is nothing to put back, and the inventory knows it.
type OnOrderRefundedHandler struct {
	inventories InventoryRepository
	tickets     TicketRepository
	clock       Clock
}

// NewOnOrderRefundedHandler wires the step.
func NewOnOrderRefundedHandler(inventories InventoryRepository, tickets TicketRepository, clock Clock) *OnOrderRefundedHandler {
	return &OnOrderRefundedHandler{inventories: inventories, tickets: tickets, clock: clock}
}

// Handle voids the tickets, then returns the seats: one transaction each.
func (h *OnOrderRefundedHandler) Handle(ctx context.Context, cmd OnOrderRefunded) error {
	showID, err := domain.ParseShowID(cmd.ShowID)
	if err != nil {
		return fmt.Errorf("on order refunded: %w", err)
	}
	orderID, err := domain.ParseOrderID(cmd.OrderID)
	if err != nil {
		return fmt.Errorf("on order refunded: %w", err)
	}
	tickets, err := h.tickets.ListByOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("on order refunded: %w", err)
	}
	for _, t := range tickets {
		t.Void(h.clock.Now())
		if err := h.tickets.Save(ctx, t); err != nil {
			return fmt.Errorf("on order refunded: void %s: %w", t.Code(), err)
		}
	}
	err = RetryOnConflict(ctx, conflictAttempts, func(ctx context.Context) error {
		inv, err := h.inventories.Get(ctx, showID, cmd.Section)
		if err != nil {
			return fmt.Errorf("load: %w", err)
		}
		inv.ReturnSeats(orderID, h.clock.Now())
		if err := h.inventories.Save(ctx, inv); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("on order refunded: %w", err)
	}
	return nil
}
