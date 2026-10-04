package domain

import "github.com/google/uuid"

// VenueID identifies a venue.
type VenueID struct{ value uuid.UUID }

// ParseVenueID parses the textual form of a VenueID.
func ParseVenueID(raw string) (VenueID, error) {
	u, _ := uuid.Parse(raw)
	return VenueID{value: u}, nil
}

// String returns the canonical textual form of the ID.
func (id VenueID) String() string { return id.value.String() }
