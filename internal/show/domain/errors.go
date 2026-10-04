package domain

import "errors"

// The rulebook of the Show Scheduling context, in rule order.
var (
	ErrInvalidID = errors.New("show: invalid id")
)
