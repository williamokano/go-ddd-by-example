package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// ConfirmHoldStep is the saga step that sells a paid order's seats.
type ConfirmHoldStep interface {
	Handle(context.Context, ConfirmHold) error
}

// IssueTicketsStep is the saga step that issues the tickets.
type IssueTicketsStep interface {
	Handle(context.Context, IssueTickets) error
}

// RefundOrderStep is the saga's compensation.
type RefundOrderStep interface {
	Handle(context.Context, RefundOrder) error
}

// CheckoutOrchestrator runs the checkout as a process manager (9.4): each
// saga fact goes to the order's CheckoutProcess, which decides the next
// step, and the orchestrator runs it. Compare with ADR-010's choreography,
// where each handler knows what follows it: here one aggregate does, and its
// state answers "where is this checkout?".
//
// A step runs before the process is saved: if the save fails, the fact is
// redelivered, the process decides the same step again, and the step, which
// is idempotent, does nothing new.
type CheckoutOrchestrator struct {
	processes CheckoutProcessRepository
	confirm   ConfirmHoldStep
	issue     IssueTicketsStep
	refund    RefundOrderStep
	clock     Clock
}

// NewCheckoutOrchestrator wires the process manager to the steps it drives.
func NewCheckoutOrchestrator(processes CheckoutProcessRepository, confirm ConfirmHoldStep, issue IssueTicketsStep,
	refund RefundOrderStep, clock Clock) *CheckoutOrchestrator {
	return &CheckoutOrchestrator{processes: processes, confirm: confirm, issue: issue, refund: refund, clock: clock}
}

// Confirm handles OrderPaid; it starts the process.
func (o *CheckoutOrchestrator) Confirm() ConfirmHoldStep { return orchestratedConfirm{o} }

// Issue handles SeatsSold.
func (o *CheckoutOrchestrator) Issue() IssueTicketsStep { return orchestratedIssue{o} }

// Refund handles HoldConfirmationFailed.
func (o *CheckoutOrchestrator) Refund() RefundOrderStep { return orchestratedRefund{o} }

type (
	orchestratedConfirm struct{ o *CheckoutOrchestrator }
	orchestratedIssue   struct{ o *CheckoutOrchestrator }
	orchestratedRefund  struct{ o *CheckoutOrchestrator }
)

func (s orchestratedConfirm) Handle(ctx context.Context, cmd ConfirmHold) error {
	return s.o.drive(ctx, cmd.OrderID, cmd.ShowID, (*domain.CheckoutProcess).OrderPaid, func(ctx context.Context) error {
		return s.o.confirm.Handle(ctx, cmd)
	})
}

func (s orchestratedIssue) Handle(ctx context.Context, cmd IssueTickets) error {
	return s.o.drive(ctx, cmd.OrderID, "", (*domain.CheckoutProcess).SeatsSold, func(ctx context.Context) error {
		return s.o.issue.Handle(ctx, cmd)
	})
}

func (s orchestratedRefund) Handle(ctx context.Context, cmd RefundOrder) error {
	return s.o.drive(ctx, cmd.OrderID, "", (*domain.CheckoutProcess).ConfirmationFailed, func(ctx context.Context) error {
		return s.o.refund.Handle(ctx, cmd)
	})
}

// drive loads (or, with a show, starts) the order's process, applies the
// fact, runs the step the process asks for, then saves the process.
func (o *CheckoutOrchestrator) drive(ctx context.Context, rawOrder, rawShow string,
	fact func(*domain.CheckoutProcess, time.Time) (domain.CheckoutStep, error), step func(context.Context) error) error {
	orderID, err := domain.ParseOrderID(rawOrder)
	if err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	p, err := o.processes.Get(ctx, orderID)
	if errors.Is(err, ErrCheckoutNotFound) && rawShow == "" {
		// Paid before orchestration was switched on: no process to ask, so
		// finish as choreography would.
		return step(ctx)
	}
	if errors.Is(err, ErrCheckoutNotFound) {
		showID, perr := domain.ParseShowID(rawShow)
		if perr != nil {
			return fmt.Errorf("checkout: %w", perr)
		}
		p, err = domain.StartCheckout(orderID, showID, o.clock.Now())
	}
	if err != nil {
		return fmt.Errorf("checkout %s: %w", orderID, err)
	}
	next, err := fact(p, o.clock.Now())
	if err != nil {
		return fmt.Errorf("checkout %s: %w", orderID, err)
	}
	if next == domain.StepNone {
		return nil // a redelivery: the process has moved on
	}
	if err := step(ctx); err != nil {
		return fmt.Errorf("checkout %s: step: %w", orderID, err)
	}
	if err := o.processes.Save(ctx, p); err != nil {
		return fmt.Errorf("checkout %s: save: %w", orderID, err)
	}
	return nil
}
