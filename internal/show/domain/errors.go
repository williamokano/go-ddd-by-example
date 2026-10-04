package domain

import "errors"

// The rulebook of the Show Scheduling context, in rule order.
var (
	ErrInvalidID = errors.New("show: invalid id")

	// SHW-1: a title of 1–200 characters; drafted only at an active venue.
	ErrInvalidTitle   = errors.New("show: invalid title")
	ErrVenueNotActive = errors.New("show: venue is not active")

	// SHW-2: doorsOpen ≤ startsAt < endsAt, at most 12h, start in the future.
	ErrInvalidSchedule = errors.New("show: invalid schedule")

	// SHW-3: no two non-cancelled shows overlap at the same venue.
	ErrScheduleConflict = errors.New("show: schedule conflict")

	// SHW-4: prices are positive amounts of one currency.
	ErrInvalidMoney     = errors.New("show: invalid money")
	ErrCurrencyMismatch = errors.New("show: currency mismatch")
	ErrInvalidPriceList = errors.New("show: invalid price list")

	// SHW-5: only a draft is priced or rescheduled; publishing needs prices.
	ErrShowNotDraft  = errors.New("show: show is not a draft")
	ErrShowNotPriced = errors.New("show: show has no price list")

	// SHW-6: the state machine.
	ErrInvalidShowTransition     = errors.New("show: invalid show transition")
	ErrInvalidCancellationReason = errors.New("show: invalid cancellation reason")

	// SHW-4: the price list covers every section of the venue, and nothing else.
	ErrPriceListMismatch = errors.New("show: price list does not match the venue's sections")
)
