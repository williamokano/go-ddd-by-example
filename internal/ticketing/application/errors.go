package application

import "errors"

// Application outcomes, not business rules.
var (
	// ErrInventoryNotFound means the show's inventory is not open (yet): its
	// show.published.v1 has not arrived, or the show was never published.
	ErrInventoryNotFound = errors.New("ticketing: inventory not found")

	// ErrHoldNotFound means no active hold has that ID.
	ErrHoldNotFound = errors.New("ticketing: hold not found")

	// ErrPaymentDeclined means the provider refused the charge.
	ErrPaymentDeclined = errors.New("ticketing: payment declined")

	// ErrOrderNotFound means no order has that ID.
	ErrOrderNotFound = errors.New("ticketing: order not found")

	// ErrTicketNotFound means no ticket has that code.
	ErrTicketNotFound = errors.New("ticketing: ticket not found")

	// ErrCheckoutNotFound means no checkout process exists for that order.
	ErrCheckoutNotFound = errors.New("ticketing: checkout not found")

	// ErrConcurrentModification means the aggregate changed since it was
	// loaded (ADR-011); use cases retry a few times, then give up.
	ErrConcurrentModification = errors.New("ticketing: concurrent modification")
)
