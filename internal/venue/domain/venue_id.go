package domain

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrInvalidVenueID is returned when a string is not a usable VenueID.
var ErrInvalidVenueID = errors.New("venue: invalid venue id")

// VenueID identifies a venue.
type VenueID struct{ value uuid.UUID }

// ParseVenueID parses the textual form of a VenueID.
func ParseVenueID(raw string) (VenueID, error) {
	u, err := uuid.Parse(raw)
	if err != nil {
		return VenueID{}, fmt.Errorf("%w: %q", ErrInvalidVenueID, raw)
	}
	return VenueID{value: u}, nil
}

// String returns the canonical textual form of the ID.
func (id VenueID) String() string { return id.value.String() }
