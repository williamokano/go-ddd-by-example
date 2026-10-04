package domain

import "errors"

// The rulebook of the Show Scheduling context, in rule order.
var (
	ErrInvalidID = errors.New("show: invalid id")

	// SHW-2: doorsOpen ≤ startsAt < endsAt, at most 12h, start in the future.
	ErrInvalidSchedule = errors.New("show: invalid schedule")
)
