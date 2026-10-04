package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// CloseInventory is Ticketing's command for "the show was cancelled".
type CloseInventory struct{ ShowID string }

// CloseInventoryHandler stops all sales for a cancelled show (TKT-11). It
// records InventoryClosed; voiding tickets and refunding orders follow from
// that fact, each in its own transaction.
type CloseInventoryHandler struct {
	inventories InventoryRepository
	clock       Clock
}

// NewCloseInventoryHandler wires the use case.
func NewCloseInventoryHandler(inventories InventoryRepository, clock Clock) *CloseInventoryHandler {
	return &CloseInventoryHandler{inventories: inventories, clock: clock}
}

// Handle closes every section, one transaction each. A show cancelled while
// still a draft was never published, so it has no sections: that's a no-op,
// not an error. A redelivery closes what a crash left open.
func (h *CloseInventoryHandler) Handle(ctx context.Context, cmd CloseInventory) error {
	showID, err := domain.ParseShowID(cmd.ShowID)
	if err != nil {
		return fmt.Errorf("close inventory: %w", err)
	}
	sections, err := h.inventories.ListByShow(ctx, showID)
	if err != nil {
		return fmt.Errorf("close inventory: %w", err)
	}
	for _, s := range sections {
		section := s.Section()
		err = RetryOnConflict(ctx, conflictAttempts, func(ctx context.Context) error {
			inv, err := h.inventories.Get(ctx, showID, section)
			if err != nil {
				return fmt.Errorf("load: %w", err)
			}
			inv.Close(h.clock.Now())
			if err := h.inventories.Save(ctx, inv); err != nil {
				return fmt.Errorf("save: %w", err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("close inventory: section %s: %w", section, err)
		}
	}
	return nil
}

// OnInventoryClosed is the cascade step driven by InventoryClosed.
type OnInventoryClosed struct{ ShowID string }

type refunder interface {
	Handle(context.Context, RefundOrder) error
}

// OnInventoryClosedHandler voids every ticket of the show and refunds every
// paid order (TKT-11): one aggregate per transaction, all idempotent.
type OnInventoryClosedHandler struct {
	tickets TicketRepository
	orders  OrderRepository
	refund  refunder
	clock   Clock
}

// NewOnInventoryClosedHandler wires the step; refund is the RefundOrder use case.
func NewOnInventoryClosedHandler(tickets TicketRepository, orders OrderRepository, refund refunder, clock Clock) *OnInventoryClosedHandler {
	return &OnInventoryClosedHandler{tickets: tickets, orders: orders, refund: refund, clock: clock}
}

// Handle voids and refunds. Re-running it finds nothing left to do.
func (h *OnInventoryClosedHandler) Handle(ctx context.Context, cmd OnInventoryClosed) error {
	showID, err := domain.ParseShowID(cmd.ShowID)
	if err != nil {
		return fmt.Errorf("on inventory closed: %w", err)
	}
	tickets, err := h.tickets.ListByShow(ctx, showID)
	if err != nil {
		return fmt.Errorf("on inventory closed: %w", err)
	}
	var errs []error
	for _, t := range tickets {
		if t.Status() == domain.VoidedTicket {
			continue
		}
		t.Void(h.clock.Now())
		if err := h.tickets.Save(ctx, t); err != nil {
			errs = append(errs, fmt.Errorf("void ticket %s: %w", t.ID(), err))
		}
	}
	orders, err := h.orders.ListPaidForShow(ctx, showID)
	if err != nil {
		return fmt.Errorf("on inventory closed: %w", err)
	}
	for _, o := range orders {
		if err := h.refund.Handle(ctx, RefundOrder{OrderID: o.ID().String()}); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("on inventory closed: %w", err)
	}
	return nil
}
