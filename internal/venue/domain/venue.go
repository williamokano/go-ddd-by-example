package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// maxVenueNameLength is VEN-1's limit, in characters.
const maxVenueNameLength = 120

// Venue is the aggregate root of Venue Management: a place where shows happen,
// with its seating layout.
type Venue struct {
	id      VenueID
	name    string
	address Address
	status  Status
}

// RegisterVenue registers a new venue. It starts as a Draft.
func RegisterVenue(id VenueID, name string, addr Address) (*Venue, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxVenueNameLength {
		return nil, fmt.Errorf("%w: %q must be 1 to %d characters", ErrInvalidVenueName, name, maxVenueNameLength)
	}
	return &Venue{id: id, name: name, address: addr, status: Draft}, nil
}

// ID returns the venue's identity.
func (v *Venue) ID() VenueID { return v.id }

// Name returns the venue's name.
func (v *Venue) Name() string { return v.name }

// Address returns where the venue is.
func (v *Venue) Address() Address { return v.address }

// Status returns where the venue is in its lifecycle.
func (v *Venue) Status() Status { return v.status }
