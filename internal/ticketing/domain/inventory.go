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

// SectionInventory is the aggregate root that protects "a seat is never sold
// twice" (TKT-5): every seat of one section of a show, and the section's
// active holds, in one consistency boundary (ADR-013, superseding ADR-005's
// one inventory per show). Holds in different sections never contend.
type SectionInventory struct {
	showID   ShowID
	section  string
	position int // the section's place in the layout, for listings
	startsAt time.Time
	seats    map[SeatRef]*seat
	order    []SeatRef // layout order, for stable listings
	holds    map[HoldID]Hold
	closed   bool
	soldOut  bool // SectionSoldOut already recorded
	version  int

	events sharedkernel.Events
}

type seat struct {
	accessible bool
	price      sharedkernel.Money
	state      SeatState
	holdID     HoldID
	orderID    OrderID
}

// OpenInventory opens the inventory of a published show from its layout
// snapshot: one SectionInventory per section, every seat and GA place
// priced and Available (TKT-1).
func OpenInventory(showID ShowID, layout InventoryLayout, startsAt time.Time, now time.Time) ([]*SectionInventory, error) {
	if showID.IsZero() {
		return nil, fmt.Errorf("%w: zero show id", ErrInvalidID)
	}
	if len(layout.Sections) == 0 {
		return nil, fmt.Errorf("%w: nothing to sell", ErrInvalidLayout)
	}
	sections := make([]*SectionInventory, 0, len(layout.Sections))
	seen := map[string]bool{}
	for i, s := range layout.Sections {
		code := strings.ToUpper(strings.TrimSpace(s.Code))
		if seen[code] {
			return nil, fmt.Errorf("%w: section %s twice", ErrInvalidLayout, code)
		}
		seen[code] = true
		inv, err := openSection(showID, s, i, startsAt, now)
		if err != nil {
			return nil, err
		}
		sections = append(sections, inv)
	}
	return sections, nil
}

func openSection(showID ShowID, s InventorySection, position int, startsAt, now time.Time) (*SectionInventory, error) {
	if !s.Price.IsPositive() {
		return nil, fmt.Errorf("%w: section %s has no price", ErrInvalidLayout, s.Code)
	}
	inv := &SectionInventory{showID: showID, position: position, startsAt: startsAt, seats: map[SeatRef]*seat{}, holds: map[HoldID]Hold{}}
	switch s.Kind {
	case KindSeated:
		for _, row := range s.Rows {
			for n := 1; n <= row.Seats; n++ {
				if err := inv.addSeat(s.Code, row.Label, n, s.Price); err != nil {
					return nil, err
				}
				if slices.Contains(row.Accessible, n) {
					inv.seats[inv.order[len(inv.order)-1]].accessible = true
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
	if len(inv.order) == 0 {
		return nil, fmt.Errorf("%w: section %s has nothing to sell", ErrInvalidLayout, s.Code)
	}
	inv.section = inv.order[0].Section()
	inv.events.Record(InventoryOpened{ShowID: showID, Section: inv.section, Seats: len(inv.order), At: now})
	return inv, nil
}

func (inv *SectionInventory) addSeat(section, row string, n int, price sharedkernel.Money) error {
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

// Hold claims 1–8 available seats of this section for a customer until
// now+ttl (TKT-2, TKT-4), all or nothing. A customer has at most one active
// hold in the section (TKT-3, per section since ADR-013); no holds once the
// show has started (TKT-12). Holds that lapsed but weren't swept yet
// are expired first, so their seats are free again.
func (inv *SectionInventory) Hold(id HoldID, customer CustomerID, seats []SeatRef, now time.Time, ttl time.Duration) error {
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
		if ref.Section() != inv.section {
			return fmt.Errorf("%w: %s is not in section %s", ErrHoldSpansSections, ref, inv.section)
		}
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
func (inv *SectionInventory) ReleaseHold(id HoldID, customer CustomerID, now time.Time) error {
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
// order again is a no-op, because the saga may redeliver. Selling the
// section's last available seat records SectionSoldOut, exactly once; whether
// the whole show sold out is ShowSoldOut's question (TKT-10).
func (inv *SectionInventory) ConfirmHold(id HoldID, order OrderID, now time.Time) error {
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
		inv.events.Record(SectionSoldOut{ShowID: inv.showID, Section: inv.section, At: now})
	}
	return nil
}

// RejectConfirmation records why a paid order's hold could not be confirmed
// (expired, released, inventory closed). The saga answers with a refund: the
// compensation (TKT-8). Nothing else changes.
func (inv *SectionInventory) RejectConfirmation(id HoldID, order OrderID, reason error, now time.Time) {
	inv.events.Record(HoldConfirmationFailed{ShowID: inv.showID, HoldID: id, OrderID: order, Reason: reason.Error(), At: now})
}

// Close stops all sales: active holds are released and no new hold is
// accepted (TKT-11, TKT-12). Closing twice is a no-op.
func (inv *SectionInventory) Close(now time.Time) {
	if inv.closed {
		return
	}
	var released []SeatRef
	for _, id := range inv.holdIDs() {
		h := inv.holds[id]
		inv.freeSeats(h)
		released = append(released, h.seats...)
		delete(inv.holds, id)
	}
	inv.closed = true
	inv.events.Record(InventoryClosed{ShowID: inv.showID, Section: inv.section, ReleasedSeats: released, At: now})
}

// ReturnSeats puts the seats sold to a returned order back on sale (9.5). A
// sold-out section is not sold out any more: it records SectionBackOnSale.
// Seats not sold to the order (a refund after a failed confirmation), a
// redelivery, or a closed inventory (a cancelled show): nothing happens.
func (inv *SectionInventory) ReturnSeats(order OrderID, now time.Time) {
	if inv.closed {
		return
	}
	var returned []SeatRef
	for _, ref := range inv.order {
		if s := inv.seats[ref]; s.state == Sold && s.orderID == order {
			s.state, s.orderID = Available, OrderID{}
			returned = append(returned, ref)
		}
	}
	if len(returned) == 0 {
		return
	}
	inv.events.Record(SeatsReturned{ShowID: inv.showID, OrderID: order, Seats: returned, At: now})
	if inv.soldOut {
		inv.soldOut = false
		inv.events.Record(SectionBackOnSale{ShowID: inv.showID, Section: inv.section, At: now})
	}
}

func (inv *SectionInventory) isSoldTo(order OrderID) bool {
	for _, s := range inv.seats {
		if s.state == Sold && s.orderID == order {
			return true
		}
	}
	return false
}

func (inv *SectionInventory) allSold() bool {
	for _, s := range inv.seats {
		if s.state != Sold {
			return false
		}
	}
	return true
}

// ExpireHolds frees the seats of every hold that has lapsed at now (TKT-4)
// and records HoldExpired for each. Time is passed in; nothing here sleeps.
func (inv *SectionInventory) ExpireHolds(now time.Time) {
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
func (inv *SectionInventory) holdIDs() []HoldID {
	ids := make([]HoldID, 0, len(inv.holds))
	for id := range inv.holds {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, func(a, b HoldID) int { return strings.Compare(a.String(), b.String()) })
	return ids
}

func (inv *SectionInventory) freeSeats(h Hold) {
	for _, ref := range h.seats {
		if s := inv.seats[ref]; s.state == Held && s.holdID == h.id {
			s.state, s.holdID = Available, HoldID{}
		}
	}
}

// HoldView returns a read-only copy of an active hold with its seats'
// prices, for placing an order.
func (inv *SectionInventory) HoldView(id HoldID) (HoldView, error) {
	h, ok := inv.holds[id]
	if !ok {
		return HoldView{}, fmt.Errorf("%w: %s", ErrHoldNotFound, id)
	}
	view := HoldView{HoldID: id, ShowID: inv.showID, Customer: h.customer, ExpiresAt: h.expiresAt}
	for _, ref := range h.seats {
		view.Lines = append(view.Lines, OrderLine{Seat: ref, Price: inv.seats[ref].price})
	}
	return view, nil
}

// ShowID returns the show this inventory sells.
func (inv *SectionInventory) ShowID() ShowID { return inv.showID }

// Section returns the code of the section this inventory sells.
func (inv *SectionInventory) Section() string { return inv.section }

// Position returns the section's place in the published layout.
func (inv *SectionInventory) Position() int { return inv.position }

// StartsAt returns when the show starts: no holds after that (TKT-12).
func (inv *SectionInventory) StartsAt() time.Time { return inv.startsAt }

// IsClosed reports whether sales are closed for good (TKT-11).
func (inv *SectionInventory) IsClosed() bool { return inv.closed }

// IsSoldOut reports whether every seat of the section is sold.
func (inv *SectionInventory) IsSoldOut() bool { return inv.soldOut }

// Version is the version the inventory was loaded at (ADR-011).
func (inv *SectionInventory) Version() int { return inv.version }

// Seats returns a copy of every seat, in layout order.
func (inv *SectionInventory) Seats() []SeatView {
	out := make([]SeatView, 0, len(inv.order))
	for _, ref := range inv.order {
		s := inv.seats[ref]
		out = append(out, SeatView{Ref: ref, Price: s.price, State: s.state, HoldID: s.holdID, OrderID: s.orderID, Accessible: s.accessible})
	}
	return out
}

// Holds returns the active holds.
func (inv *SectionInventory) Holds() []Hold {
	out := make([]Hold, 0, len(inv.holds))
	for _, id := range inv.holdIDs() {
		out = append(out, inv.holds[id])
	}
	return out
}

// PullEvents returns the recorded events and forgets them.
func (inv *SectionInventory) PullEvents() []sharedkernel.DomainEvent { return inv.events.PullEvents() }
