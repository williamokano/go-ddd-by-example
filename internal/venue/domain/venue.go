package domain

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// maxVenueNameLength is VEN-1's limit, in characters.
const maxVenueNameLength = 120

// Venue is the aggregate root of Venue Management: a place where shows happen,
// with its seating layout.
type Venue struct {
	id       VenueID
	name     string
	address  Address
	status   Status
	sections []Section
}

// RegisterVenue registers a new venue. It starts as a Draft.
func RegisterVenue(id VenueID, name string, addr Address) (*Venue, error) {
	if id.IsZero() {
		return nil, fmt.Errorf("%w: zero id", ErrInvalidVenueID)
	}
	if addr.IsZero() {
		return nil, fmt.Errorf("%w: zero address", ErrInvalidAddress)
	}
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

// AddSection adds a section to the venue's layout. Section codes are unique
// within the venue (VEN-2): only the root sees all the sections, so only the
// root can enforce it.
func (v *Venue) AddSection(s Section) error {
	for _, existing := range v.sections {
		if existing.Code() == s.Code() {
			return fmt.Errorf("%w: %s", ErrDuplicateSectionCode, s.Code())
		}
	}
	v.sections = append(v.sections, s)
	return nil
}

// Sections returns a copy of the venue's sections, in the order they were
// added. Handing out the internal slice would let callers bypass VEN-2 and VEN-4.
func (v *Venue) Sections() []Section { return slices.Clone(v.sections) }

// Capacity returns the venue's total number of places: the seats in seated
// sections plus the capacity of general admission sections (VEN-7).
func (v *Venue) Capacity() int {
	total := 0
	for _, s := range v.sections {
		total += s.Capacity()
	}
	return total
}
