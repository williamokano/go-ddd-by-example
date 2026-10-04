package domain_test

import (
	"errors"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// TKT-14: the buyer may return a fulfilled order until the show starts.
func TestOrder_CheckReturn(t *testing.T) {
	fulfilled := func(t *testing.T) *domain.Order {
		t.Helper()
		o := placed(t)
		ref, _ := domain.NewPaymentRef("pay_123")
		_ = o.MarkPaid(ref, now)
		_ = o.MarkFulfilled(nil, now)
		return o
	}

	if err := fulfilled(t).CheckReturn(ana, startsAt, now); err != nil {
		t.Errorf("the buyer, before the show: error = %v", err)
	}
	if err := fulfilled(t).CheckReturn(bob, startsAt, now); !errors.Is(err, domain.ErrNotOrderOwner) {
		t.Errorf("someone else: error = %v, want %v", err, domain.ErrNotOrderOwner)
	}
	if err := fulfilled(t).CheckReturn(ana, startsAt, startsAt); !errors.Is(err, domain.ErrSalesClosed) {
		t.Errorf("once the show started: error = %v, want %v", err, domain.ErrSalesClosed)
	}
	if err := placed(t).CheckReturn(ana, startsAt, now); !errors.Is(err, domain.ErrInvalidOrderTransition) {
		t.Errorf("an unpaid order: error = %v, want %v", err, domain.ErrInvalidOrderTransition)
	}
}

func TestOrder_RefundedCarriesItsSeats(t *testing.T) {
	o := placed(t)
	ref, _ := domain.NewPaymentRef("pay_123")
	_ = o.MarkPaid(ref, now)
	o.PullEvents()

	_ = o.MarkRefunded(now)

	refunded := o.PullEvents()[0].(domain.OrderRefunded)
	if refunded.Section != "FLOOR" || len(refunded.Seats) != 2 {
		t.Errorf("OrderRefunded = %+v, want FLOOR and its 2 seats", refunded)
	}
}

// A returned order's seats are on sale again; a sold-out section is not any
// more. Seats that were never sold to the order stay as they are.
func TestSectionInventory_ReturnSeats(t *testing.T) {
	inv := orch(t)
	order := newOrderID()
	if err := inv.ConfirmHold(held(t, inv, ana, "ORCH/A/1", "ORCH/A/2", "ORCH/B/1"), order, now); err != nil {
		t.Fatal(err)
	}
	inv.PullEvents()

	inv.ReturnSeats(order, now)
	inv.ReturnSeats(order, now) // redelivered
	inv.ReturnSeats(newOrderID(), now)

	if s := states(inv); s["ORCH/A/1"] != "available" || s["ORCH/B/1"] != "available" || inv.IsSoldOut() {
		t.Errorf("states %v, sold out %v; want available, not sold out", s, inv.IsSoldOut())
	}
	if names := eventNames(inv); len(names) != 2 || names[0] != "ticketing.SeatsReturned" || names[1] != "ticketing.SectionBackOnSale" {
		t.Errorf("events = %v, want SeatsReturned then SectionBackOnSale, once", names)
	}
}

func TestSectionInventory_ReturnSeats_OnAClosedInventoryIsANoOp(t *testing.T) {
	inv := orch(t)
	order := newOrderID()
	_ = inv.ConfirmHold(held(t, inv, ana, "ORCH/A/1"), order, now)
	inv.Close(now)
	inv.PullEvents()

	inv.ReturnSeats(order, now)

	if states(inv)["ORCH/A/1"] != "sold" || len(inv.PullEvents()) != 0 {
		t.Error("a cancelled show's seats went back on sale")
	}
}
