package domain_test

import (
	"errors"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// The checkout as a process manager (9.4): one aggregate knows where every
// checkout is and decides the next step; the steps stay the same use cases.
func TestCheckoutProcess(t *testing.T) {
	order := newOrderID()
	start := func(t *testing.T) *domain.CheckoutProcess {
		t.Helper()
		p, err := domain.StartCheckout(order, showID, now)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}

	t.Run("paid → confirm the seats; sold → issue the tickets; done", func(t *testing.T) {
		p := start(t)

		if step, err := p.OrderPaid(now); err != nil || step != domain.StepConfirmSeats {
			t.Fatalf("OrderPaid = %v, %v; want ConfirmSeats", step, err)
		}
		if step, err := p.SeatsSold(now); err != nil || step != domain.StepIssueTickets {
			t.Fatalf("SeatsSold = %v, %v; want IssueTickets", step, err)
		}
		if p.State() != domain.CheckoutCompleted {
			t.Errorf("state = %v, want completed", p.State())
		}
	})

	t.Run("confirmation failed → refund: the compensation", func(t *testing.T) {
		p := start(t)
		_, _ = p.OrderPaid(now)

		step, err := p.ConfirmationFailed(now)

		if err != nil || step != domain.StepRefund || p.State() != domain.CheckoutCompensated {
			t.Errorf("ConfirmationFailed = %v, %v, state %v; want Refund, compensated", step, err, p.State())
		}
	})

	t.Run("a redelivered fact asks for nothing", func(t *testing.T) {
		p := start(t)
		_, _ = p.OrderPaid(now)
		_, _ = p.SeatsSold(now)

		paid, err1 := p.OrderPaid(now)
		sold, err2 := p.SeatsSold(now)

		if err := errors.Join(err1, err2); err != nil || paid != domain.StepNone || sold != domain.StepNone {
			t.Errorf("redelivery = %v, %v, %v; want no steps", paid, sold, err)
		}
	})

	t.Run("a fact out of order is an invalid transition", func(t *testing.T) {
		if _, err := start(t).SeatsSold(now); !errors.Is(err, domain.ErrInvalidCheckoutTransition) {
			t.Errorf("error = %v, want %v", err, domain.ErrInvalidCheckoutTransition)
		}
	})

	t.Run("records each advance", func(t *testing.T) {
		p := start(t)
		p.PullEvents()
		_, _ = p.OrderPaid(now)

		ev := p.PullEvents()
		want := domain.CheckoutAdvanced{OrderID: order, From: domain.CheckoutStarted, To: domain.CheckoutAwaitingSeats, At: now}
		if len(ev) != 1 || ev[0] != want {
			t.Errorf("events = %v, want %v", ev, want)
		}
	})
}
