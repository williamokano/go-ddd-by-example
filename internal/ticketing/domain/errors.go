package domain

import "errors"

// The rulebook of Ticketing, in rule order.
var (
	ErrInvalidID      = errors.New("ticketing: invalid id")
	ErrInvalidSeatRef = errors.New("ticketing: invalid seat reference")
)
