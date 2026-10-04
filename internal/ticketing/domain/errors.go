package domain

import "errors"

// The rulebook of Ticketing, in rule order.
var (
	ErrInvalidID      = errors.New("ticketing: invalid id")
	ErrInvalidSeatRef = errors.New("ticketing: invalid seat reference")
	ErrInvalidLayout  = errors.New("ticketing: invalid inventory layout")

	// TKT-2: a hold has 1–8 distinct, existing, available seats, all or nothing.
	ErrInvalidHoldSize = errors.New("ticketing: a hold has 1 to 8 seats")
	ErrDuplicateSeat   = errors.New("ticketing: the same seat twice")
	ErrUnknownSeat     = errors.New("ticketing: no such seat")
	ErrSeatUnavailable = errors.New("ticketing: seat unavailable") // TKT-5

	// TKT-3: one active hold per customer per show.
	ErrCustomerAlreadyHolding = errors.New("ticketing: customer already holds seats for this show")

	// TKT-4, TKT-8: only a live hold, by its owner, can be released or confirmed.
	ErrHoldNotFound = errors.New("ticketing: hold not found")
	ErrNotHoldOwner = errors.New("ticketing: the hold belongs to another customer")
	ErrHoldExpired  = errors.New("ticketing: hold expired")

	// TKT-6: an order needs a valid contact email (and a live, owned hold).
	ErrInvalidContactEmail = errors.New("ticketing: invalid contact email")

	// TKT-7: the order lifecycle.
	ErrInvalidOrderTransition = errors.New("ticketing: invalid order transition")

	// TKT-13: a ticket is checked in once, never when voided, on the show's day.
	ErrAlreadyCheckedIn = errors.New("ticketing: ticket already checked in")
	ErrTicketVoided     = errors.New("ticketing: ticket is voided")
	ErrNotShowDay       = errors.New("ticketing: not the day of the show")

	// ADR-013: a hold is for seats of one section.
	ErrHoldSpansSections = errors.New("ticketing: a hold is for one section")

	// TKT-12: no new holds on a closed inventory or after the show's start.
	ErrSalesClosed = errors.New("ticketing: sales are closed")
)
