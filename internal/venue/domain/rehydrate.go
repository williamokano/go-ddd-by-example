package domain

import "slices"

// VenueState is everything a repository stores about a venue.
type VenueState struct {
	ID       VenueID
	Name     string
	Address  Address
	Status   Status
	Sections []Section
	Version  int
}

// RehydrateVenue rebuilds a venue from storage. It is reconstitution, not
// creation: it trusts the state (it was valid when it was saved), runs no
// business rules and records no events, because nothing new happened.
//
// Only driven adapters (repositories) may call it; the architecture test
// enforces that from Part 4.
func RehydrateVenue(s VenueState) *Venue {
	return &Venue{
		id:       s.ID,
		name:     s.Name,
		address:  s.Address,
		status:   s.Status,
		sections: slices.Clone(s.Sections),
		version:  s.Version,
	}
}
