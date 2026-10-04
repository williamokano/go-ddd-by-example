package domain

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ErrInvalidVenueID is returned when a string is not a usable VenueID.
var ErrInvalidVenueID = errors.New("venue: invalid venue id")

// VenueID identifies a venue. The field is unexported, so outside this package
// a VenueID can only come from NewVenueID or ParseVenueID (or be the zero value).
type VenueID struct{ value uuid.UUID }

// NewVenueID wraps an already-generated UUID. The application's IDGenerator
// calls it; the domain never generates IDs itself (ADR-008).
func NewVenueID(u uuid.UUID) VenueID { return VenueID{value: u} }

// ParseVenueID parses the textual form of a VenueID. The nil UUID is rejected:
// an ID that identifies nothing is not an ID.
func ParseVenueID(raw string) (VenueID, error) {
	u, err := uuid.Parse(raw)
	if err != nil || u == uuid.Nil {
		return VenueID{}, fmt.Errorf("%w: %q", ErrInvalidVenueID, raw)
	}
	return VenueID{value: u}, nil
}

// String returns the canonical textual form of the ID.
func (id VenueID) String() string { return id.value.String() }

// IsZero reports whether the ID is the zero value, i.e. no ID at all.
func (id VenueID) IsZero() bool { return id.value == uuid.Nil }
