package application_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/payment/fakegateway"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

type sagaFixture struct {
	*fixture
	orders   *memory.OrderRepository
	tickets  *memory.TicketRepository
	gateway  *fakegateway.Gateway
	checkout *application.CheckoutHandler
	confirm  *application.ConfirmHoldHandler
	issue    *application.IssueTicketsHandler
	refund   *application.RefundOrderHandler
}

func newSaga(t *testing.T, mode fakegateway.Mode) *sagaFixture {
	t.Helper()
	f := newFixture(t)
	orders, tickets, gateway := memory.NewOrderRepository(), memory.NewTicketRepository(), fakegateway.New(mode)
	return &sagaFixture{
		fixture: f, orders: orders, tickets: tickets, gateway: gateway,
		checkout: application.NewCheckoutHandler(f.inventories, orders, gateway, f.ids, domain.StandardPricing(), f.clock),
		confirm:  application.NewConfirmHoldHandler(f.inventories, f.clock),
		issue:    application.NewIssueTicketsHandler(orders, tickets, f.ids, f.clock),
		refund:   application.NewRefundOrderHandler(orders, gateway, f.clock),
	}
}

// heldSeats opens a show and holds ORCH/A/1 + ORCH/A/2 for a new customer.
func (f *sagaFixture) heldSeats(t *testing.T) (show, customer string, hold domain.HoldID) {
	t.Helper()
	show, customer = f.openShow(t), uuid.NewString()
	return show, customer, f.holdSeats(t, show, customer, "ORCH/A/1", "ORCH/A/2")
}

func (f *sagaFixture) order(t *testing.T, id domain.OrderID) *domain.Order {
	t.Helper()
	o, err := f.orders.Get(f.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestCheckout(t *testing.T) {
	t.Run("an approved payment leaves the order Paid and OrderPaid saved", func(t *testing.T) {
		f := newSaga(t, fakegateway.Mode{})
		_, customer, hold := f.heldSeats(t)

		res, err := f.checkout.Handle(f.ctx, application.Checkout{HoldID: hold.String(), CustomerID: customer, ContactEmail: "Ana@Example.com"})

		if err != nil {
			t.Fatal(err)
		}
		if o := f.order(t, res.OrderID); o.Status() != domain.Paid || o.Total().String() != "EUR 104.94" { // 90.00 + 10% fee + 6% VAT (TKT-15)
			t.Errorf("order = %v %v", o.Status(), o.Total())
		}
		if !published[domain.OrderPaid](f.orders.Published()) {
			t.Error("OrderPaid was not saved")
		}
	})

	t.Run("a declined payment leaves PaymentFailed, and the hold stays", func(t *testing.T) {
		f := newSaga(t, fakegateway.Mode{Decline: true})
		show, customer, hold := f.heldSeats(t)

		res, err := f.checkout.Handle(f.ctx, application.Checkout{HoldID: hold.String(), CustomerID: customer, ContactEmail: "ana@example.com"})

		if err != nil || f.order(t, res.OrderID).Status() != domain.PaymentFailed {
			t.Errorf("err = %v, status %v", err, f.order(t, res.OrderID).Status())
		}
		if f.seatStates(t, show)["ORCH/A/1"] != "held" {
			t.Error("the hold was lost: the customer can no longer retry")
		}
	})

	t.Run("a malformed email fails before any order exists (TKT-6)", func(t *testing.T) {
		f := newSaga(t, fakegateway.Mode{})
		_, customer, hold := f.heldSeats(t)

		_, err := f.checkout.Handle(f.ctx, application.Checkout{HoldID: hold.String(), CustomerID: customer, ContactEmail: "nope"})

		if !errors.Is(err, domain.ErrInvalidContactEmail) || len(f.orders.Published()) != 0 || len(f.gateway.Charges()) != 0 {
			t.Errorf("err = %v, %d events, %d charges", err, len(f.orders.Published()), len(f.gateway.Charges()))
		}
	})

	t.Run("someone else's hold is refused (TKT-6)", func(t *testing.T) {
		f := newSaga(t, fakegateway.Mode{})
		_, _, hold := f.heldSeats(t)

		_, err := f.checkout.Handle(f.ctx, application.Checkout{HoldID: hold.String(), CustomerID: uuid.NewString(), ContactEmail: "eve@example.com"})

		if !errors.Is(err, domain.ErrNotHoldOwner) {
			t.Errorf("error = %v, want %v", err, domain.ErrNotHoldOwner)
		}
	})
}

// paidOrder checks out a hold with an approving gateway.
func (f *sagaFixture) paidOrder(t *testing.T) (show string, hold domain.HoldID, order domain.OrderID) {
	t.Helper()
	show, customer, hold := f.heldSeats(t)
	res, err := f.checkout.Handle(f.ctx, application.Checkout{HoldID: hold.String(), CustomerID: customer, ContactEmail: "ana@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	return show, hold, res.OrderID
}

func TestConfirmHold(t *testing.T) {
	t.Run("sells the seats; a redelivery is a no-op (TKT-8)", func(t *testing.T) {
		f := newSaga(t, fakegateway.Mode{})
		show, hold, order := f.paidOrder(t)
		cmd := application.ConfirmHold{ShowID: show, Section: "ORCH", HoldID: hold.String(), OrderID: order.String()}

		if err := f.confirm.Handle(f.ctx, cmd); err != nil {
			t.Fatal(err)
		}
		if err := f.confirm.Handle(f.ctx, cmd); err != nil {
			t.Fatal(err)
		}

		if s := f.seatStates(t, show); s["ORCH/A/1"] != "sold" || s["ORCH/A/2"] != "sold" {
			t.Errorf("seats = %v", s)
		}
		if n := count[domain.SeatsSold](f.inventories.Published()); n != 1 {
			t.Errorf("SeatsSold recorded %d times, want 1", n)
		}
	})

	t.Run("an expired hold records HoldConfirmationFailed (TKT-8)", func(t *testing.T) {
		f := newSaga(t, fakegateway.Mode{})
		show, hold, order := f.paidOrder(t)
		f.clock.Advance(holdTTL + time.Second)

		err := f.confirm.Handle(f.ctx, application.ConfirmHold{ShowID: show, Section: "ORCH", HoldID: hold.String(), OrderID: order.String()})

		if err != nil {
			t.Fatal(err)
		}
		if f.seatStates(t, show)["ORCH/A/1"] == "sold" || !published[domain.HoldConfirmationFailed](f.inventories.Published()) {
			t.Error("want the seats unsold and HoldConfirmationFailed recorded")
		}
	})
}

func TestIssueTickets(t *testing.T) {
	f := newSaga(t, fakegateway.Mode{})
	show, _, order := f.paidOrder(t)
	cmd := application.IssueTickets{ShowID: show, OrderID: order.String(), Seats: []string{"ORCH/A/1", "ORCH/A/2"}}

	if err := f.issue.Handle(f.ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if err := f.issue.Handle(f.ctx, cmd); err != nil { // redelivered
		t.Fatal(err)
	}

	tickets, _ := f.tickets.ListByOrder(f.ctx, order)
	if len(tickets) != 2 || tickets[0].Code() == tickets[1].Code() {
		t.Errorf("%d tickets; want 2 with unique codes (TKT-9)", len(tickets))
	}
	if f.order(t, order).Status() != domain.Fulfilled {
		t.Errorf("order = %v, want fulfilled", f.order(t, order).Status())
	}
}

func TestRefundOrder(t *testing.T) {
	f := newSaga(t, fakegateway.Mode{})
	_, _, order := f.paidOrder(t)

	if err := f.refund.Handle(f.ctx, application.RefundOrder{OrderID: order.String()}); err != nil {
		t.Fatal(err)
	}
	if err := f.refund.Handle(f.ctx, application.RefundOrder{OrderID: order.String()}); err != nil { // redelivered
		t.Fatal(err)
	}

	if f.order(t, order).Status() != domain.Refunded || len(f.gateway.Refunds()) != 1 {
		t.Errorf("status %v, %d refunds; want refunded once (the compensation)", f.order(t, order).Status(), len(f.gateway.Refunds()))
	}
}

// TKT-10 across sections (ADR-013): the show is announced sold out only once
// the last section sells out.
func TestOnSectionSoldOut(t *testing.T) {
	f := newSaga(t, fakegateway.Mode{})
	policy := application.NewOnSectionSoldOutHandler(f.inventories, f.inventories, f.clock)
	show := f.openShow(t)
	sell := func(section string, seats ...string) {
		t.Helper()
		customer := uuid.NewString()
		hold := f.holdSeats(t, show, customer, seats...)
		res, err := f.checkout.Handle(f.ctx, application.Checkout{HoldID: hold.String(), CustomerID: customer, ContactEmail: "ana@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		cmd := application.ConfirmHold{ShowID: show, Section: section, HoldID: hold.String(), OrderID: res.OrderID.String()}
		if err := f.confirm.Handle(f.ctx, cmd); err != nil {
			t.Fatal(err)
		}
		if err := policy.Handle(f.ctx, application.OnSectionSoldOut{ShowID: show}); err != nil {
			t.Fatal(err)
		}
	}

	sell("ORCH", "ORCH/A/1", "ORCH/A/2")
	if published[domain.InventorySoldOut](f.inventories.Published()) {
		t.Fatal("InventorySoldOut with the FLOOR still on sale")
	}

	sell("FLOOR", "FLOOR/GA/0001", "FLOOR/GA/0002", "FLOOR/GA/0003")
	if n := count[domain.InventorySoldOut](f.inventories.Published()); n != 1 {
		t.Errorf("InventorySoldOut published %d times, want 1", n)
	}
	if n := count[domain.SectionSoldOut](f.inventories.Published()); n != 2 {
		t.Errorf("SectionSoldOut recorded %d times, want 2", n)
	}
}

// TKT-14 → 9.5: the buyer returns a fulfilled order; the saga puts the seats
// back and voids the tickets. The sold-out section is back on sale.
func TestReturnOrder_PutsTheShowBackOnSale(t *testing.T) {
	f := newSaga(t, fakegateway.Mode{})
	show := f.openShow(t)
	buy := func(section string, seats ...string) (string, domain.OrderID) {
		t.Helper()
		customer := uuid.NewString()
		hold := f.holdSeats(t, show, customer, seats...)
		res, err := f.checkout.Handle(f.ctx, application.Checkout{HoldID: hold.String(), CustomerID: customer, ContactEmail: "ana@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		confirm := application.ConfirmHold{ShowID: show, Section: section, HoldID: hold.String(), OrderID: res.OrderID.String()}
		if err := f.confirm.Handle(f.ctx, confirm); err != nil {
			t.Fatal(err)
		}
		if err := f.issue.Handle(f.ctx, application.IssueTickets{ShowID: show, OrderID: res.OrderID.String(), Seats: seats}); err != nil {
			t.Fatal(err)
		}
		return customer, res.OrderID
	}
	customer, orchOrder := buy("ORCH", "ORCH/A/1", "ORCH/A/2")
	buy("FLOOR", "FLOOR/GA/0001", "FLOOR/GA/0002", "FLOOR/GA/0003")
	returnOrder := application.NewReturnOrderHandler(f.orders, memory.NewShowSchedule(f.inventories), f.refund, f.clock)
	returned := application.NewOnOrderRefundedHandler(f.inventories, f.tickets, f.clock)

	if err := returnOrder.Handle(f.ctx, application.ReturnOrder{OrderID: orchOrder.String(), CustomerID: customer}); err != nil {
		t.Fatal(err)
	}
	if err := returned.Handle(f.ctx, application.OnOrderRefunded{ShowID: show, Section: "ORCH", OrderID: orchOrder.String()}); err != nil {
		t.Fatal(err)
	}

	if f.order(t, orchOrder).Status() != domain.Refunded || f.seatStates(t, show)["ORCH/A/1"] != "available" {
		t.Errorf("order %v, ORCH/A/1 %s; want refunded and available", f.order(t, orchOrder).Status(), f.seatStates(t, show)["ORCH/A/1"])
	}
	tickets, _ := f.tickets.ListByOrder(f.ctx, orchOrder)
	if len(tickets) != 2 || tickets[0].Status() != domain.VoidedTicket {
		t.Errorf("tickets = %d, first %v; want 2 voided", len(tickets), tickets[0].Status())
	}
	if n := count[domain.SectionBackOnSale](f.inventories.Published()); n != 1 {
		t.Errorf("SectionBackOnSale recorded %d times, want 1", n)
	}
}

func TestReturnOrder_OnlyByTheBuyer(t *testing.T) {
	f := newSaga(t, fakegateway.Mode{})
	_, _, order := f.paidOrder(t)
	returnOrder := application.NewReturnOrderHandler(f.orders, memory.NewShowSchedule(f.inventories), f.refund, f.clock)

	err := returnOrder.Handle(f.ctx, application.ReturnOrder{OrderID: order.String(), CustomerID: uuid.NewString()})

	if !errors.Is(err, domain.ErrNotOrderOwner) {
		t.Errorf("error = %v, want %v (TKT-14)", err, domain.ErrNotOrderOwner)
	}
}

// TKT-15 through the use case: the venue's country reaches the price.
func TestCheckout_PricesWithFeeAndVAT(t *testing.T) {
	f := newSaga(t, fakegateway.Mode{})
	_, _, order := f.paidOrder(t) // ORCH/A/1 + A/2 at EUR 45, in Portugal

	got := f.order(t, order).Pricing()

	if got.Subtotal.Amount() != 9000 || got.Fee.Amount() != 900 || got.VAT.Amount() != 594 || got.Total.Amount() != 10494 {
		t.Errorf("pricing = %v, want 90.00 + 9.00 fee + 5.94 VAT", got)
	}
	view, err := application.NewOrderQueries(f.orders, f.tickets).Get(f.ctx, order)
	if err != nil || view.Amount != 10494 || view.Subtotal != 9000 || view.Fee != 900 || view.VAT != 594 {
		t.Errorf("view = %+v, %v", view, err)
	}
}
