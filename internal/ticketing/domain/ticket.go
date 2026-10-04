package domain

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
)

// TicketCode is what's printed on a ticket and checked at the gate (TKT-9).
type TicketCode struct{ value string }

// TicketCodeFor derives a ticket's code from its ID: "ABCD-EFGH-IJKL". Same ID,
// same code; different IDs, different codes (in practice: 60 bits).
func TicketCodeFor(id TicketID) TicketCode {
	sum := sha256.Sum256(id.value[:])
	raw := base32.StdEncoding.EncodeToString(sum[:])[:12]
	return TicketCode{value: raw[0:4] + "-" + raw[4:8] + "-" + raw[8:12]}
}

// NewTicketCode wraps a stored code.
func NewTicketCode(raw string) (TicketCode, error) {
	if len(raw) != 14 {
		return TicketCode{}, fmt.Errorf("%w: ticket code %q", ErrInvalidID, raw)
	}
	return TicketCode{value: raw}, nil
}

// String returns the code.
func (c TicketCode) String() string { return c.value }

// TicketStatus is Valid, CheckedIn or Voided.
type TicketStatus uint8

// Ticket statuses.
const (
	ValidTicket TicketStatus = iota + 1
	VoidedTicket
	CheckedInTicket
)

// String returns the stored name.
func (s TicketStatus) String() string {
	switch s {
	case VoidedTicket:
		return "voided"
	case CheckedInTicket:
		return "checked_in"
	default:
		return "valid"
	}
}

// GateID names the entrance a ticket was scanned at.
type GateID struct{ value string }

// NewGateID validates a gate's name.
func NewGateID(raw string) (GateID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return GateID{}, fmt.Errorf("%w: blank gate", ErrInvalidID)
	}
	return GateID{value: raw}, nil
}

// String returns the gate's name.
func (g GateID) String() string { return g.value }

// checkInWindow is TKT-13's "on the day of the show": Ticketing knows when
// the show starts, not the venue's time zone, so the day is the 24 hours
// around the start.
const checkInWindow = 12 * time.Hour

// Ticket is proof of entry for one seat at one show. A small aggregate of its
// own, so voiding or checking it in locks nothing else.
type Ticket struct {
	id      TicketID
	code    TicketCode
	showID  ShowID
	orderID OrderID
	seat    SeatRef
	status  TicketStatus
	version int

	checkedInAt time.Time
	gate        GateID

	events sharedkernel.Events
}

// IssueTicket issues the ticket for one sold seat (TKT-9).
func IssueTicket(id TicketID, show ShowID, order OrderID, seat SeatRef, now time.Time) (*Ticket, error) {
	if id.IsZero() || show.IsZero() || order.IsZero() {
		return nil, fmt.Errorf("%w: zero ticket, show or order id", ErrInvalidID)
	}
	t := &Ticket{id: id, code: TicketCodeFor(id), showID: show, orderID: order, seat: seat, status: ValidTicket}
	t.events.Record(TicketIssued{TicketID: id, Code: t.code, ShowID: show, OrderID: order, Seat: seat, At: now})
	return t, nil
}

// Void invalidates the ticket (the show was cancelled, TKT-11). Voiding twice
// is a no-op.
func (t *Ticket) Void(now time.Time) {
	if t.status == VoidedTicket {
		return
	}
	t.status = VoidedTicket
	t.events.Record(TicketVoided{TicketID: t.id, ShowID: t.showID, At: now})
}

// CheckIn lets the ticket in through gate (TKT-13): once, never when voided,
// and only on the day of the show, which starts at startsAt.
func (t *Ticket) CheckIn(gate GateID, startsAt, now time.Time) error {
	switch t.status {
	case VoidedTicket:
		return fmt.Errorf("%w: %s", ErrTicketVoided, t.code)
	case CheckedInTicket:
		return fmt.Errorf("%w: %s at %s through %s", ErrAlreadyCheckedIn, t.code, t.checkedInAt.Format(time.RFC3339), t.gate)
	}
	if now.Before(startsAt.Add(-checkInWindow)) || now.After(startsAt.Add(checkInWindow)) {
		return fmt.Errorf("%w: the show starts at %s", ErrNotShowDay, startsAt.Format(time.RFC3339))
	}
	t.status, t.checkedInAt, t.gate = CheckedInTicket, now, gate
	t.events.Record(TicketCheckedIn{TicketID: t.id, ShowID: t.showID, Gate: gate, At: now})
	return nil
}

// ID returns the ticket's identity.
func (t *Ticket) ID() TicketID { return t.id }

// Code returns the ticket code.
func (t *Ticket) Code() TicketCode { return t.code }

// ShowID returns the show.
func (t *Ticket) ShowID() ShowID { return t.showID }

// OrderID returns the order that bought it.
func (t *Ticket) OrderID() OrderID { return t.orderID }

// Seat returns the seat.
func (t *Ticket) Seat() SeatRef { return t.seat }

// Status returns Valid or Voided.
func (t *Ticket) Status() TicketStatus { return t.status }

// CheckedInAt returns when the ticket was let in, or the zero time.
func (t *Ticket) CheckedInAt() time.Time { return t.checkedInAt }

// Gate returns the gate it was let in through, or the zero GateID.
func (t *Ticket) Gate() GateID { return t.gate }

// Version is the version the ticket was loaded at.
func (t *Ticket) Version() int { return t.version }

// PullEvents returns the recorded events and forgets them.
func (t *Ticket) PullEvents() []sharedkernel.DomainEvent { return t.events.PullEvents() }

// TicketState is what a repository stores about a ticket.
type TicketState struct {
	ID      TicketID
	Code    TicketCode
	ShowID  ShowID
	OrderID OrderID
	Seat    SeatRef
	Status  TicketStatus
	Version int

	CheckedInAt time.Time
	Gate        GateID
}

// TicketStateOf reads a ticket out for storage.
func TicketStateOf(t *Ticket) TicketState {
	return TicketState{
		ID: t.id, Code: t.code, ShowID: t.showID, OrderID: t.orderID, Seat: t.seat, Status: t.status, Version: t.version,
		CheckedInAt: t.checkedInAt, Gate: t.gate,
	}
}

// RehydrateTicket rebuilds a ticket from storage: no rules, no events.
func RehydrateTicket(s TicketState) *Ticket {
	return &Ticket{
		id: s.ID, code: s.Code, showID: s.ShowID, orderID: s.OrderID, seat: s.Seat, status: s.Status, version: s.Version,
		checkedInAt: s.CheckedInAt, gate: s.Gate,
	}
}
