// Package domain is the Show Scheduling domain model: shows, their schedule
// and price list, the lifecycle Draft → Published → SoldOut / Cancelled, the
// scheduling policy (SHW-3), and Show's own read-only copy of venue facts.
//
// It imports only the standard library and github.com/google/uuid. It never
// imports the venue context: Show's VenueID and VenueLayout are its own types.
package domain
