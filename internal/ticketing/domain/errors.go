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

	// TKT-12: no new holds on a closed inventory or after the show's start.
	ErrSalesClosed = errors.New("ticketing: sales are closed")
)
