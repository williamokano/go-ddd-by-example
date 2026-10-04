package application

import "errors"

// Storage outcomes, not business rules: they belong to the application layer.
var (
	// ErrVenueNotFound: no venue has the requested ID.
	ErrVenueNotFound = errors.New("venue: not found")

	// ErrConcurrentModification: the venue changed since it was loaded
	// (optimistic concurrency, ADR-011). The caller may retry load-decide-save.
	ErrConcurrentModification = errors.New("venue: concurrent modification")
)
