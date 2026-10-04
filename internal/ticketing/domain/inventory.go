package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
)

// maxSeatsPerHold is TKT-2's limit.
const maxSeatsPerHold = 8

// ShowInventory is the aggregate root that protects "a seat is never sold
// twice" (TKT-5): every seat of one show and every active hold, in one
// consistency boundary (ADR-005).
type ShowInventory struct {
	showID   ShowID
	startsAt time.Time
	seats    map[SeatRef]*seat
	order    []SeatRef // layout order, for stable listings
	holds    map[HoldID]Hold
	closed   bool
	soldOut  bool // InventorySoldOut already recorded (TKT-10)
	version  int

	events sharedkernel.Events
}

type seat struct {
	price   sharedkernel.Money
	state   SeatState
	holdID  HoldID
	orderID OrderID
}

// OpenInventory opens the inventory of a published show from its layout
// snapshot: every seat and GA place, priced, Available (TKT-1).
func OpenInventory(showID ShowID, layout InventoryLayout, startsAt time.Time, now time.Time) (*ShowInventory, error) {
	if showID.IsZero() {
		return nil, fmt.Errorf("%w: zero show id", ErrInvalidID)
	}
	inv := &ShowInventory{showID: showID, startsAt: startsAt, seats: map[SeatRef]*seat{}, holds: map[HoldID]Hold{}}
	for _, s := range layout.Sections {
		if !s.Price.IsPositive() {
			return nil, fmt.Errorf("%w: section %s has no price", ErrInvalidLayout, s.Code)
		}
		switch s.Kind {
		case KindSeated:
			for _, row := range s.Rows {
				for n := 1; n <= row.Seats; n++ {
					if err := inv.addSeat(s.Code, row.Label, n, s.Price); err != nil {
						return nil, err
					}
				}
			}
		case KindGA:
			for n := 1; n <= s.Capacity; n++ {
				if err := inv.addSeat(s.Code, gaRow, n, s.Price); err != nil {
					return nil, err
				}
			}
		default:
			return nil, fmt.Errorf("%w: section %s has kind %q", ErrInvalidLayout, s.Code, s.Kind)
		}
	}
	if len(inv.order) == 0 {
		return nil, fmt.Errorf("%w: nothing to sell", ErrInvalidLayout)
	}
	inv.events.Record(InventoryOpened{ShowID: showID, Seats: len(inv.order), At: now})
	return inv, nil
}

func (inv *ShowInventory) addSeat(section, row string, n int, price sharedkernel.Money) error {
	ref, err := NewSeatRef(section, row, n)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidLayout, err)
	}
	if _, dup := inv.seats[ref]; dup {
		return fmt.Errorf("%w: seat %s twice", ErrInvalidLayout, ref)
	}
	inv.seats[ref] = &seat{price: price, state: Available}
	inv.order = append(inv.order, ref)
	return nil
}

// Hold claims 1–8 available seats for a customer until now+ttl (TKT-2, TKT-4),
// all or nothing. A customer has at most one active hold (TKT-3); no holds
// once the show has started (TKT-12). Holds that lapsed but weren't swept yet
// are expired first, so their seats are free again.
func (inv *ShowInventory) Hold(id HoldID, customer CustomerID, seats []SeatRef, now time.Time, ttl time.Duration) error {
	if id.IsZero() || customer.IsZero() {
		return fmt.Errorf("%w: zero hold or customer id", ErrInvalidID)
	}
	if inv.closed {
		return fmt.Errorf("%w: the inventory is closed", ErrSalesClosed)
	}
	if !now.Before(inv.startsAt) {
		return fmt.Errorf("%w: the show starts at %s", ErrSalesClosed, inv.startsAt.Format(time.RFC3339))
	}
	if len(seats) == 0 || len(seats) > maxSeatsPerHold {
		return fmt.Errorf("%w: %d seats, want 1 to %d", ErrInvalidHoldSize, len(seats), maxSeatsPerHold)
	}
	seen := make(map[SeatRef]bool, len(seats))
	for _, ref := range seats {
		if seen[ref] {
			return fmt.Errorf("%w: %s", ErrDuplicateSeat, ref)
		}
		seen[ref] = true
		if _, ok := inv.seats[ref]; !ok {
			return fmt.Errorf("%w: %s", ErrUnknownSeat, ref)
		}
	}

	inv.ExpireHolds(now)

	for _, h := range inv.holds {
		if h.customer == customer {
			return fmt.Errorf("%w: hold %s", ErrCustomerAlreadyHolding, h.id)
		}
	}
	for _, ref := range seats {
		if s := inv.seats[ref]; s.state != Available {
			return fmt.Errorf("%w: %s is %s", ErrSeatUnavailable, ref, s.state)
		}
	}

	for _, ref := range seats {
		s := inv.seats[ref]
		s.state, s.holdID = Held, id
	}
	h := Hold{id: id, customer: customer, seats: slices.Clone(seats), expiresAt: now.Add(ttl)}
	inv.holds[id] = h
	inv.events.Record(SeatsHeld{ShowID: inv.showID, HoldID: id, CustomerID: customer, Seats: h.Seats(), ExpiresAt: h.expiresAt, At: now})
	return nil
}

// ReleaseHold gives a customer's held seats back.
func (inv *ShowInventory) ReleaseHold(id HoldID, customer CustomerID, now time.Time) error {
	h, ok := inv.holds[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrHoldNotFound, id)
	}
	if h.customer != customer {
		return fmt.Errorf("%w: hold %s", ErrNotHoldOwner, id)
	}
	inv.freeSeats(h)
	delete(inv.holds, id)
	inv.events.Record(HoldReleased{ShowID: inv.showID, HoldID: id, Seats: h.Seats(), At: now})
	return nil
}

// ConfirmHold sells the seats of a paid order's hold (TKT-8). A hold that
// lapsed, even if not swept yet, or that was released, can't be confirmed:
// the order must be refunded (the saga's compensation). Confirming the same
// order again is a no-op, because the saga may redeliver. Selling the last
// available seat records InventorySoldOut, exactly once (TKT-10).
func (inv *ShowInventory) ConfirmHold(id HoldID, order OrderID, now time.Time) error {
	if inv.isSoldTo(order) {
		return nil
	}
	h, ok := inv.holds[id]
	if !ok {
		return fmt.Errorf("%w: %s", ErrHoldNotFound, id)
	}
	if h.IsExpired(now) {
		return fmt.Errorf("%w: hold %s lapsed at %s", ErrHoldExpired, id, h.expiresAt.Format(time.RFC3339))
	}
	for _, ref := range h.seats {
		s := inv.seats[ref]
		s.state, s.holdID, s.orderID = Sold, HoldID{}, order
	}
	delete(inv.holds, id)
	inv.events.Record(SeatsSold{ShowID: inv.showID, HoldID: id, OrderID: order, Seats: h.Seats(), At: now})
	if !inv.soldOut && inv.allSold() {
		inv.soldOut = true
		inv.events.Record(InventorySoldOut{ShowID: inv.showID, At: now})
	}
	return nil
}

// Close stops all sales: active holds are released and no new hold is
// accepted (TKT-11, TKT-12). Closing twice is a no-op.
func (inv *ShowInventory) Close(now time.Time) {
	if inv.closed {
		return
	}
	for _, id := range inv.holdIDs() {
		inv.freeSeats(inv.holds[id])
		delete(inv.holds, id)
	}
	inv.closed = true
	inv.events.Record(InventoryClosed{ShowID: inv.showID, At: now})
}

func (inv *ShowInventory) isSoldTo(order OrderID) bool {
	for _, s := range inv.seats {
		if s.state == Sold && s.orderID == order {
			return true
		}
	}
	return false
}

func (inv *ShowInventory) allSold() bool {
	for _, s := range inv.seats {
		if s.state != Sold {
			return false
		}
	}
	return true
}

// ExpireHolds frees the seats of every hold that has lapsed at now (TKT-4)
// and records HoldExpired for each. Time is passed in; nothing here sleeps.
func (inv *ShowInventory) ExpireHolds(now time.Time) {
	for _, id := range inv.holdIDs() {
		if h := inv.holds[id]; h.IsExpired(now) {
			inv.freeSeats(h)
			delete(inv.holds, id)
			inv.events.Record(HoldExpired{ShowID: inv.showID, HoldID: id, Seats: h.Seats(), At: now})
		}
	}
}

// holdIDs lists the active holds in a stable order, so events are recorded
// deterministically.
func (inv *ShowInventory) holdIDs() []HoldID {
	ids := make([]HoldID, 0, len(inv.holds))
	for id := range inv.holds {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b HoldID) int { return strings.Compare(a.String(), b.String()) })
	return ids
}

func (inv *ShowInventory) freeSeats(h Hold) {
	for _, ref := range h.seats {
		if s := inv.seats[ref]; s.state == Held && s.holdID == h.id {
			s.state, s.holdID = Available, HoldID{}
		}
	}
}

// ShowID returns the show this inventory sells.
func (inv *ShowInventory) ShowID() ShowID { return inv.showID }

// StartsAt returns when the show starts: no holds after that (TKT-12).
func (inv *ShowInventory) StartsAt() time.Time { return inv.startsAt }

// IsClosed reports whether sales are closed for good (TKT-11).
func (inv *ShowInventory) IsClosed() bool { return inv.closed }

// IsSoldOut reports whether InventorySoldOut was recorded.
func (inv *ShowInventory) IsSoldOut() bool { return inv.soldOut }

// Version is the version the inventory was loaded at (ADR-011).
func (inv *ShowInventory) Version() int { return inv.version }

// Seats returns a copy of every seat, in layout order.
func (inv *ShowInventory) Seats() []SeatView {
	out := make([]SeatView, 0, len(inv.order))
	for _, ref := range inv.order {
		s := inv.seats[ref]
		out = append(out, SeatView{Ref: ref, Price: s.price, State: s.state, HoldID: s.holdID, OrderID: s.orderID})
	}
	return out
}

// Holds returns the active holds.
func (inv *ShowInventory) Holds() []Hold {
	out := make([]Hold, 0, len(inv.holds))
	for _, id := range inv.holdIDs() {
		out = append(out, inv.holds[id])
	}
	return out
}

// PullEvents returns the recorded events and forgets them.
func (inv *ShowInventory) PullEvents() []sharedkernel.DomainEvent { return inv.events.PullEvents() }
