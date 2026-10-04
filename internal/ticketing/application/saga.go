package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// ConfirmHold is the saga step driven by OrderPaid.
type ConfirmHold struct {
	ShowID  string
	HoldID  string
	OrderID string
}

// ConfirmHoldHandler sells a paid order's held seats (tx3, on the inventory).
type ConfirmHoldHandler struct {
	inventories InventoryRepository
	clock       Clock
}

// NewConfirmHoldHandler wires the step.
func NewConfirmHoldHandler(inventories InventoryRepository, clock Clock) *ConfirmHoldHandler {
	return &ConfirmHoldHandler{inventories: inventories, clock: clock}
}

// Handle confirms the hold. If it lapsed or is gone, the inventory records
// HoldConfirmationFailed instead and the saga compensates with a refund
// (TKT-8). Redelivery is a no-op: the domain knows the order is confirmed.
func (h *ConfirmHoldHandler) Handle(ctx context.Context, cmd ConfirmHold) error {
	showID, err := domain.ParseShowID(cmd.ShowID)
	if err != nil {
		return fmt.Errorf("confirm hold: %w", err)
	}
	holdID, err := domain.ParseHoldID(cmd.HoldID)
	if err != nil {
		return fmt.Errorf("confirm hold: %w", err)
	}
	orderID, err := domain.ParseOrderID(cmd.OrderID)
	if err != nil {
		return fmt.Errorf("confirm hold: %w", err)
	}
	err = RetryOnConflict(ctx, conflictAttempts, func(ctx context.Context) error {
		inv, err := h.inventories.Get(ctx, showID)
		if err != nil {
			return fmt.Errorf("load: %w", err)
		}
		now := h.clock.Now()
		if err := inv.ConfirmHold(holdID, orderID, now); err != nil {
			if !errors.Is(err, domain.ErrHoldExpired) && !errors.Is(err, domain.ErrHoldNotFound) {
				return err
			}
			inv.RejectConfirmation(holdID, orderID, err, now)
		}
		if err := h.inventories.Save(ctx, inv); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("confirm hold: %w", err)
	}
	return nil
}

// IssueTickets is the saga step driven by SeatsSold.
type IssueTickets struct {
	ShowID  string
	OrderID string
	Seats   []string
}

// IssueTicketsHandler issues one ticket per sold seat and fulfils the order
// (tx4). It creates several fresh Ticket aggregates in one go: nobody else
// can see them yet, so there is no contention to protect against.
type IssueTicketsHandler struct {
	orders  OrderRepository
	tickets TicketRepository
	ids     IDGenerator
	clock   Clock
}

// NewIssueTicketsHandler wires the step.
func NewIssueTicketsHandler(orders OrderRepository, tickets TicketRepository, ids IDGenerator, clock Clock) *IssueTicketsHandler {
	return &IssueTicketsHandler{orders: orders, tickets: tickets, ids: ids, clock: clock}
}

// Handle issues the tickets (TKT-9). Ticket IDs derive from order + seat, so
// a redelivered SeatsSold issues no duplicates.
func (h *IssueTicketsHandler) Handle(ctx context.Context, cmd IssueTickets) error {
	showID, err := domain.ParseShowID(cmd.ShowID)
	if err != nil {
		return fmt.Errorf("issue tickets: %w", err)
	}
	orderID, err := domain.ParseOrderID(cmd.OrderID)
	if err != nil {
		return fmt.Errorf("issue tickets: %w", err)
	}
	var issued []domain.IssuedTicket
	for _, raw := range cmd.Seats {
		seat, err := domain.ParseSeatRef(raw)
		if err != nil {
			return fmt.Errorf("issue tickets: %w", err)
		}
		ticket, err := domain.IssueTicket(h.ids.TicketIDFor(orderID, seat), showID, orderID, seat, h.clock.Now())
		if err != nil {
			return fmt.Errorf("issue tickets: %w", err)
		}
		if err := h.tickets.Save(ctx, ticket); err != nil {
			return fmt.Errorf("issue tickets: %w", err)
		}
		issued = append(issued, domain.IssuedTicket{Seat: seat, Code: ticket.Code()})
	}
	err = RetryOnConflict(ctx, conflictAttempts, func(ctx context.Context) error {
		order, err := h.orders.Get(ctx, orderID)
		if err != nil {
			return fmt.Errorf("load: %w", err)
		}
		if err := order.MarkFulfilled(issued, h.clock.Now()); err != nil {
			return err
		}
		if err := h.orders.Save(ctx, order); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("issue tickets: %w", err)
	}
	return nil
}

// RefundOrder is the compensation driven by HoldConfirmationFailed (and, in
// 7.10, by a cancelled show).
type RefundOrder struct{ OrderID string }

// RefundOrderHandler gives a paid order's money back (tx5).
type RefundOrderHandler struct {
	orders  OrderRepository
	gateway PaymentGateway
	clock   Clock
}

// NewRefundOrderHandler wires the step.
func NewRefundOrderHandler(orders OrderRepository, gateway PaymentGateway, clock Clock) *RefundOrderHandler {
	return &RefundOrderHandler{orders: orders, gateway: gateway, clock: clock}
}

// Handle refunds the order once: a refunded order is left alone.
func (h *RefundOrderHandler) Handle(ctx context.Context, cmd RefundOrder) error {
	orderID, err := domain.ParseOrderID(cmd.OrderID)
	if err != nil {
		return fmt.Errorf("refund order: %w", err)
	}
	err = RetryOnConflict(ctx, conflictAttempts, func(ctx context.Context) error {
		order, err := h.orders.Get(ctx, orderID)
		if err != nil {
			return fmt.Errorf("load: %w", err)
		}
		if order.Status() == domain.Refunded {
			return nil
		}
		if order.Status() != domain.Paid && order.Status() != domain.Fulfilled {
			return nil // nothing was charged: nothing to give back
		}
		if err := h.gateway.Refund(ctx, order.PaymentRef(), order.Total()); err != nil {
			return fmt.Errorf("refund: %w", err)
		}
		if err := order.MarkRefunded(h.clock.Now()); err != nil {
			return err
		}
		if err := h.orders.Save(ctx, order); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("refund order: %w", err)
	}
	return nil
}
