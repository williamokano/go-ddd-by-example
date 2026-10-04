package domain

import (
	"fmt"
	"slices"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
)

// OrderStatus is where an order is in its lifecycle (TKT-7).
type OrderStatus uint8

// The order lifecycle: Pending → Paid → Fulfilled, Pending → PaymentFailed,
// Paid or Fulfilled → Refunded.
const (
	Pending OrderStatus = iota + 1
	Paid
	PaymentFailed
	Fulfilled
	Refunded
)

var orderStatusNames = map[OrderStatus]string{
	Pending: "pending", Paid: "paid", PaymentFailed: "payment_failed", Fulfilled: "fulfilled", Refunded: "refunded",
}

// String returns the stored name of the status.
func (s OrderStatus) String() string {
	if name, ok := orderStatusNames[s]; ok {
		return name
	}
	return "unknown"
}

// OrderLine is one seat of an order, at the price it was sold for.
type OrderLine struct {
	Seat  SeatRef
	Price sharedkernel.Money
}

// HoldView is a read-only copy of a hold with its seats' prices: what an
// order needs to know about the inventory, without touching it.
type HoldView struct {
	HoldID    HoldID
	ShowID    ShowID
	Country   string // the venue's, for VAT (TKT-15)
	Customer  CustomerID
	ExpiresAt time.Time
	Lines     []OrderLine
}

// Order is the aggregate root of one checkout of one hold (TKT-6, TKT-7).
type Order struct {
	id         OrderID
	showID     ShowID
	holdID     HoldID
	customer   CustomerID
	email      ContactEmail
	lines      []OrderLine
	pricing    PriceBreakdown
	status     OrderStatus
	paymentRef PaymentRef
	version    int

	events sharedkernel.Events
}

// PlaceOrder starts a checkout for the customer's own, live hold (TKT-6),
// priced by the PricingPolicy for the venue's country (TKT-15): the order
// records the breakdown, the policy decides it.
func PlaceOrder(id OrderID, customer CustomerID, email ContactEmail, hold HoldView, pricing PricingPolicy, now time.Time) (*Order, error) {
	if id.IsZero() || customer.IsZero() {
		return nil, fmt.Errorf("%w: zero order or customer id", ErrInvalidID)
	}
	if email == (ContactEmail{}) {
		return nil, fmt.Errorf("%w: no contact email", ErrInvalidContactEmail)
	}
	if hold.Customer != customer {
		return nil, fmt.Errorf("%w: hold %s", ErrNotHoldOwner, hold.HoldID)
	}
	if !now.Before(hold.ExpiresAt) {
		return nil, fmt.Errorf("%w: hold %s", ErrHoldExpired, hold.HoldID)
	}
	if len(hold.Lines) == 0 {
		return nil, fmt.Errorf("%w: hold %s has no seats", ErrHoldNotFound, hold.HoldID)
	}
	breakdown, err := pricing.Price(hold.Lines, hold.Country)
	if err != nil {
		return nil, err
	}
	o := &Order{
		id: id, showID: hold.ShowID, holdID: hold.HoldID, customer: customer, email: email,
		lines: slices.Clone(hold.Lines), pricing: breakdown, status: Pending,
	}
	o.events.Record(OrderPlaced{OrderID: id, ShowID: hold.ShowID, HoldID: hold.HoldID, CustomerID: customer, Total: breakdown.Total, At: now})
	return o, nil
}

// MarkPaid records the payment provider's successful charge.
func (o *Order) MarkPaid(ref PaymentRef, now time.Time) error {
	switch o.status {
	case Paid:
		return nil
	case Pending:
		o.status, o.paymentRef = Paid, ref
		o.events.Record(OrderPaid{OrderID: o.id, ShowID: o.showID, Section: o.lines[0].Seat.Section(), HoldID: o.holdID, At: now})
		return nil
	default:
		return o.illegal("mark paid")
	}
}

// MarkPaymentFailed records a declined charge. The hold stays: the customer
// may try again with a new order.
func (o *Order) MarkPaymentFailed(reason string, now time.Time) error {
	switch o.status {
	case PaymentFailed:
		return nil
	case Pending:
		o.status = PaymentFailed
		o.events.Record(OrderPaymentFailed{OrderID: o.id, Reason: reason, At: now})
		return nil
	default:
		return o.illegal("mark payment failed")
	}
}

// IssuedTicket is a ticket as the order reports it.
type IssuedTicket struct {
	Seat SeatRef
	Code TicketCode
}

// MarkFulfilled records that the tickets were issued (TKT-9). The event
// carries the tickets and the contact email: Notifications sends them.
func (o *Order) MarkFulfilled(tickets []IssuedTicket, now time.Time) error {
	switch o.status {
	case Fulfilled:
		return nil
	case Paid:
		o.status = Fulfilled
		o.events.Record(OrderFulfilled{
			OrderID: o.id, ShowID: o.showID, CustomerID: o.customer, ContactEmail: o.email,
			Tickets: slices.Clone(tickets), At: now,
		})
		return nil
	default:
		return o.illegal("mark fulfilled")
	}
}

// MarkRefunded records that the money went back: compensation when the hold
// could not be confirmed (TKT-8), or a cancelled show (TKT-11).
func (o *Order) MarkRefunded(now time.Time) error {
	switch o.status {
	case Refunded:
		return nil
	case Paid, Fulfilled:
		o.status = Refunded
		seats := make([]SeatRef, len(o.lines))
		for i, l := range o.lines {
			seats[i] = l.Seat
		}
		o.events.Record(OrderRefunded{
			OrderID: o.id, ShowID: o.showID, Section: o.lines[0].Seat.Section(), Seats: seats,
			CustomerID: o.customer, ContactEmail: o.email, Total: o.pricing.Total, At: now,
		})
		return nil
	default:
		return o.illegal("refund")
	}
}

// CheckReturn says whether customer may return the order (TKT-14): only the
// buyer, only a fulfilled order, only before the show starts at startsAt.
// The refund itself is the saga's RefundOrder step.
func (o *Order) CheckReturn(customer CustomerID, startsAt, now time.Time) error {
	if customer != o.customer {
		return fmt.Errorf("%w: order %s", ErrNotOrderOwner, o.id)
	}
	if o.status != Fulfilled {
		return o.illegal("return")
	}
	if !now.Before(startsAt) {
		return fmt.Errorf("%w: the show started at %s", ErrSalesClosed, startsAt.Format(time.RFC3339))
	}
	return nil
}

func (o *Order) illegal(what string) error {
	return fmt.Errorf("%w: cannot %s a %s order", ErrInvalidOrderTransition, what, o.status)
}

// ID returns the order's identity.
func (o *Order) ID() OrderID { return o.id }

// ShowID returns the show the order is for.
func (o *Order) ShowID() ShowID { return o.showID }

// HoldID returns the hold the order checks out.
func (o *Order) HoldID() HoldID { return o.holdID }

// Customer returns who placed the order.
func (o *Order) Customer() CustomerID { return o.customer }

// ContactEmail returns where tickets and notices go.
func (o *Order) ContactEmail() ContactEmail { return o.email }

// Lines returns a copy of the order's lines.
func (o *Order) Lines() []OrderLine { return slices.Clone(o.lines) }

// Total returns what the customer pays: lines, fee and VAT (TKT-15).
func (o *Order) Total() sharedkernel.Money { return o.pricing.Total }

// Pricing returns the breakdown of the total.
func (o *Order) Pricing() PriceBreakdown { return o.pricing }

// Status returns where the order is in its lifecycle.
func (o *Order) Status() OrderStatus { return o.status }

// PaymentRef returns the provider's charge reference (zero until paid).
func (o *Order) PaymentRef() PaymentRef { return o.paymentRef }

// Version is the version the order was loaded at (ADR-011).
func (o *Order) Version() int { return o.version }

// PullEvents returns the recorded events and forgets them.
func (o *Order) PullEvents() []sharedkernel.DomainEvent { return o.events.PullEvents() }

// OrderState is everything a repository stores about an order.
type OrderState struct {
	ID         OrderID
	ShowID     ShowID
	HoldID     HoldID
	Customer   CustomerID
	Email      ContactEmail
	Lines      []OrderLine
	Pricing    PriceBreakdown
	Status     OrderStatus
	PaymentRef PaymentRef
	Version    int
}

// OrderStateOf reads an order out for storage.
func OrderStateOf(o *Order) OrderState {
	return OrderState{
		ID: o.id, ShowID: o.showID, HoldID: o.holdID, Customer: o.customer, Email: o.email,
		Lines: o.Lines(), Pricing: o.pricing, Status: o.status, PaymentRef: o.paymentRef, Version: o.version,
	}
}

// RehydrateOrder rebuilds an order from storage: no rules, no events.
func RehydrateOrder(s OrderState) *Order {
	return &Order{
		id: s.ID, showID: s.ShowID, holdID: s.HoldID, customer: s.Customer, email: s.Email,
		lines: slices.Clone(s.Lines), pricing: s.Pricing, status: s.Status, paymentRef: s.PaymentRef, version: s.Version,
	}
}
