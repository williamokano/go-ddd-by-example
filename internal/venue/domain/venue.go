package domain

import "strings"

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
	return &Venue{id: id, name: strings.TrimSpace(name), address: addr, status: Draft}, nil
}

// ID returns the venue's identity.
func (v *Venue) ID() VenueID { return v.id }

// Name returns the venue's name.
func (v *Venue) Name() string { return v.name }

// Address returns where the venue is.
func (v *Venue) Address() Address { return v.address }

// Status returns where the venue is in its lifecycle.
func (v *Venue) Status() Status { return v.status }
