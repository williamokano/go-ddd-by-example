// Package application holds the Venue Management use cases: one handler per
// user intention (RegisterVenue, AddSection, ActivateVenue, RetireVenue) and
// the query side (GetVenue, ListVenues).
//
// It also declares every driven port the core needs (repositories, clock,
// ID generator, queries) as interfaces. Adapters implement them; this package
// never imports an adapter, a driver or the platform. It contains no business
// rules: those live in the domain.
package application
