package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
)

// ErrInvalidCheckoutTransition means a fact arrived that the checkout's state
// does not expect, such as SeatsSold before OrderPaid.
var ErrInvalidCheckoutTransition = errors.New("ticketing: invalid checkout transition")

// CheckoutState is where a checkout is.
type CheckoutState uint8

// Checkout states.
const (
	CheckoutStarted CheckoutState = iota + 1
	CheckoutAwaitingSeats
	CheckoutCompleted
	CheckoutCompensated
)

var checkoutStateNames = map[CheckoutState]string{
	CheckoutStarted: "started", CheckoutAwaitingSeats: "awaiting_seats",
	CheckoutCompleted: "completed", CheckoutCompensated: "compensated",
}

// String returns the stored name.
func (s CheckoutState) String() string { return checkoutStateNames[s] }

// ParseCheckoutState reads a stored name.
func ParseCheckoutState(raw string) (CheckoutState, error) {
	for s, name := range checkoutStateNames {
		if name == raw {
			return s, nil
		}
	}
	return 0, fmt.Errorf("%w: checkout state %q", ErrInvalidCheckoutTransition, raw)
}

// CheckoutStep is the command the process asks for next.
type CheckoutStep uint8

// Checkout steps.
const (
	StepNone CheckoutStep = iota
	StepConfirmSeats
	StepIssueTickets
	StepRefund
)

// CheckoutProcess is the checkout saga as a process manager (9.4): an
// aggregate per order that knows where the checkout is and decides each next
// step. The steps themselves stay the same use cases; what moves here is the
// flow that choreography (ADR-010) spreads across the handlers.
type CheckoutProcess struct {
	orderID OrderID
	showID  ShowID
	state   CheckoutState
	version int

	events sharedkernel.Events
}

// StartCheckout starts the process for a placed order.
func StartCheckout(order OrderID, show ShowID, now time.Time) (*CheckoutProcess, error) {
	if order.IsZero() || show.IsZero() {
		return nil, fmt.Errorf("%w: zero order or show id", ErrInvalidID)
	}
	return &CheckoutProcess{orderID: order, showID: show, state: CheckoutStarted}, nil
}

// OrderPaid: the money is in, so confirm the held seats.
func (p *CheckoutProcess) OrderPaid(now time.Time) (CheckoutStep, error) {
	return p.advance(CheckoutStarted, CheckoutAwaitingSeats, StepConfirmSeats, now)
}

// SeatsSold: the seats are the order's, so issue the tickets. That is the
// last step: the checkout is complete.
func (p *CheckoutProcess) SeatsSold(now time.Time) (CheckoutStep, error) {
	return p.advance(CheckoutAwaitingSeats, CheckoutCompleted, StepIssueTickets, now)
}

// ConfirmationFailed: the hold was gone, so give the money back (TKT-8).
func (p *CheckoutProcess) ConfirmationFailed(now time.Time) (CheckoutStep, error) {
	return p.advance(CheckoutAwaitingSeats, CheckoutCompensated, StepRefund, now)
}

// advance moves from → to and asks for step. A fact for a state already
// left is a redelivery: no step. Any other state is out of order.
func (p *CheckoutProcess) advance(from, to CheckoutState, step CheckoutStep, now time.Time) (CheckoutStep, error) {
	switch {
	case p.state == from:
		p.state = to
		p.events.Record(CheckoutAdvanced{OrderID: p.orderID, From: from, To: to, At: now})
		return step, nil
	case p.state > from:
		return StepNone, nil
	default:
		return StepNone, fmt.Errorf("%w: %s, expected %s", ErrInvalidCheckoutTransition, p.state, from)
	}
}

// OrderID returns the order the process drives.
func (p *CheckoutProcess) OrderID() OrderID { return p.orderID }

// ShowID returns the order's show.
func (p *CheckoutProcess) ShowID() ShowID { return p.showID }

// State returns where the checkout is.
func (p *CheckoutProcess) State() CheckoutState { return p.state }

// Version is the version the process was loaded at.
func (p *CheckoutProcess) Version() int { return p.version }

// PullEvents returns the recorded events and forgets them.
func (p *CheckoutProcess) PullEvents() []sharedkernel.DomainEvent { return p.events.PullEvents() }

// CheckoutProcessState is what a repository stores about a process.
type CheckoutProcessState struct {
	OrderID OrderID
	ShowID  ShowID
	State   CheckoutState
	Version int
}

// CheckoutProcessStateOf reads a process out for storage.
func CheckoutProcessStateOf(p *CheckoutProcess) CheckoutProcessState {
	return CheckoutProcessState{OrderID: p.orderID, ShowID: p.showID, State: p.state, Version: p.version}
}

// RehydrateCheckoutProcess rebuilds a process from storage.
func RehydrateCheckoutProcess(s CheckoutProcessState) *CheckoutProcess {
	return &CheckoutProcess{orderID: s.OrderID, showID: s.ShowID, state: s.State, version: s.Version}
}

// CheckoutAdvanced records a checkout moving on. Internal: the process's
// own log, not a message.
type CheckoutAdvanced struct {
	OrderID  OrderID
	From, To CheckoutState
	At       time.Time
}

// EventName implements DomainEvent.
func (CheckoutAdvanced) EventName() string { return "ticketing.CheckoutAdvanced" }

// OccurredAt implements DomainEvent.
func (e CheckoutAdvanced) OccurredAt() time.Time { return e.At }
