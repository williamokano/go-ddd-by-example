package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
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

	// events is a named field, not embedded: embedding would promote Record
	// onto *Venue and let any caller fake the venue's history.
	events Events
}

// RegisterVenue registers a new venue. It starts as a Draft.
func RegisterVenue(id VenueID, name string, addr Address, now time.Time) (*Venue, error) {
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
	v := &Venue{id: id, name: name, address: addr, status: Draft}
	v.events.Record(VenueRegistered{VenueID: id, Name: name, At: now})
	return v, nil
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
// root can enforce it. The layout can only change while the venue is a
// draft (VEN-4).
func (v *Venue) AddSection(s Section, now time.Time) error {
	if v.status != Draft {
		return fmt.Errorf("%w: venue is %s", ErrVenueNotDraft, v.status)
	}
	if s.IsZero() {
		return fmt.Errorf("%w: zero section", ErrInvalidSection)
	}
	for _, existing := range v.sections {
		if existing.Code() == s.Code() {
			return fmt.Errorf("%w: %s", ErrDuplicateSectionCode, s.Code())
		}
	}
	v.sections = append(v.sections, s)
	v.events.Record(SectionAdded{VenueID: v.id, Section: s, At: now})
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

// Activate opens the venue for shows. Only a draft venue with at least one
// section can be activated (VEN-5).
func (v *Venue) Activate(now time.Time) error {
	if v.status != Draft {
		return fmt.Errorf("%w: cannot activate a %s venue", ErrInvalidVenueTransition, v.status)
	}
	if len(v.sections) == 0 {
		return ErrVenueHasNoSections
	}
	v.status = Active
	v.events.Record(VenueActivated{VenueID: v.id, Name: v.name, Sections: v.Sections(), At: now})
	return nil
}

// Retire closes the venue for good. Only an active venue can be retired, and
// Retired is terminal (VEN-6).
func (v *Venue) Retire(now time.Time) error {
	if v.status != Active {
		return fmt.Errorf("%w: cannot retire a %s venue", ErrInvalidVenueTransition, v.status)
	}
	v.status = Retired
	v.events.Record(VenueRetired{VenueID: v.id, At: now})
	return nil
}

// PullEvents returns the events recorded since the last pull, in order, and
// forgets them. The repository calls it after saving, to fill the outbox.
func (v *Venue) PullEvents() []DomainEvent { return v.events.PullEvents() }
