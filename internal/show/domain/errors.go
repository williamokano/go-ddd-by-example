package domain

import "errors"

// The rulebook of the Show Scheduling context, in rule order.
var (
	ErrInvalidID = errors.New("show: invalid id")

	// SHW-2: doorsOpen ≤ startsAt < endsAt, at most 12h, start in the future.
	ErrInvalidSchedule = errors.New("show: invalid schedule")

	// SHW-4: prices are positive amounts of one currency.
	ErrInvalidMoney     = errors.New("show: invalid money")
	ErrCurrencyMismatch = errors.New("show: currency mismatch")
	ErrInvalidPriceList = errors.New("show: invalid price list")

	// SHW-4: the price list covers every section of the venue, and nothing else.
	ErrPriceListMismatch = errors.New("show: price list does not match the venue's sections")
)
