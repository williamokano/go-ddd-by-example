package application

import "errors"

// Application outcomes, not business rules.
var (
	// ErrShowNotFound means no show has the requested ID.
	ErrShowNotFound = errors.New("show: not found")

	// ErrConcurrentModification means the show changed since it was loaded.
	ErrConcurrentModification = errors.New("show: concurrent modification")

	// ErrVenueUnknown means Show has not (yet) heard of the venue: its
	// venue.activated.v1 has not arrived. Eventually consistent, so retry later.
	ErrVenueUnknown = errors.New("show: unknown venue")

	// ErrNotPromoter means the caller is not the promoter who drafted the show.
	ErrNotPromoter = errors.New("show: only the show's promoter may do that")
)
