package application_test

import (
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/payment/fakegateway"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// The same checkout, orchestrated (9.4): the saga consumer hands each fact to
// the CheckoutOrchestrator, which asks the process what to do and runs that
// step. The steps are the very handlers choreography calls directly.
func TestCheckoutOrchestrator(t *testing.T) {
	setup := func(t *testing.T) (*sagaFixture, *memory.CheckoutProcessRepository, *application.CheckoutOrchestrator) {
		t.Helper()
		f := newSaga(t, fakegateway.Mode{})
		processes := memory.NewCheckoutProcessRepository()
		return f, processes, application.NewCheckoutOrchestrator(processes, f.confirm, f.issue, f.refund, f.clock)
	}
	state := func(t *testing.T, processes *memory.CheckoutProcessRepository, f *sagaFixture, order domain.OrderID) domain.CheckoutState {
		t.Helper()
		p, err := processes.Get(f.ctx, order)
		if err != nil {
			t.Fatal(err)
		}
		return p.State()
	}

	t.Run("paid, confirmed, ticketed: completed", func(t *testing.T) {
		f, processes, o := setup(t)
		show, hold, order := f.paidOrder(t)

		if err := o.Confirm().Handle(f.ctx, application.ConfirmHold{ShowID: show, Section: "ORCH", HoldID: hold.String(), OrderID: order.String()}); err != nil {
			t.Fatal(err)
		}
		if s := f.seatStates(t, show); s["ORCH/A/1"] != "sold" || state(t, processes, f, order) != domain.CheckoutAwaitingSeats {
			t.Fatalf("seats %v, process %v", s, state(t, processes, f, order))
		}
		if err := o.Issue().Handle(f.ctx, application.IssueTickets{ShowID: show, OrderID: order.String(), Seats: []string{"ORCH/A/1", "ORCH/A/2"}}); err != nil {
			t.Fatal(err)
		}

		if f.order(t, order).Status() != domain.Fulfilled || state(t, processes, f, order) != domain.CheckoutCompleted {
			t.Errorf("order %v, process %v; want fulfilled, completed", f.order(t, order).Status(), state(t, processes, f, order))
		}
	})

	t.Run("a lapsed hold: compensated with a refund", func(t *testing.T) {
		f, processes, o := setup(t)
		show, hold, order := f.paidOrder(t)
		f.clock.Advance(holdTTL + time.Second)

		if err := o.Confirm().Handle(f.ctx, application.ConfirmHold{ShowID: show, Section: "ORCH", HoldID: hold.String(), OrderID: order.String()}); err != nil {
			t.Fatal(err)
		}
		if err := o.Refund().Handle(f.ctx, application.RefundOrder{OrderID: order.String()}); err != nil {
			t.Fatal(err)
		}

		if f.order(t, order).Status() != domain.Refunded || state(t, processes, f, order) != domain.CheckoutCompensated {
			t.Errorf("order %v, process %v; want refunded, compensated", f.order(t, order).Status(), state(t, processes, f, order))
		}
	})

	t.Run("a redelivered fact runs no step twice", func(t *testing.T) {
		f, _, o := setup(t)
		show, hold, order := f.paidOrder(t)
		cmd := application.ConfirmHold{ShowID: show, Section: "ORCH", HoldID: hold.String(), OrderID: order.String()}

		for range 2 {
			if err := o.Confirm().Handle(f.ctx, cmd); err != nil {
				t.Fatal(err)
			}
		}

		if n := count[domain.SeatsSold](f.inventories.Published()); n != 1 {
			t.Errorf("SeatsSold %d times, want 1", n)
		}
	})

	t.Run("an order paid before orchestration finishes as choreography would", func(t *testing.T) {
		f, _, o := setup(t)
		show, hold, order := f.paidOrder(t)
		if err := f.confirm.Handle(f.ctx, application.ConfirmHold{ShowID: show, Section: "ORCH", HoldID: hold.String(), OrderID: order.String()}); err != nil {
			t.Fatal(err) // confirmed by the choreographed saga: no process exists
		}

		err := o.Issue().Handle(f.ctx, application.IssueTickets{ShowID: show, OrderID: order.String(), Seats: []string{"ORCH/A/1", "ORCH/A/2"}})

		if err != nil || f.order(t, order).Status() != domain.Fulfilled {
			t.Errorf("err = %v, order %v; want fulfilled", err, f.order(t, order).Status())
		}
	})
}
