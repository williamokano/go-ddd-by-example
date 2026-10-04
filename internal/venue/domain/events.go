package domain

import "time"

// VenueRegistered records that a venue manager registered a new venue.
type VenueRegistered struct {
	VenueID VenueID
	Name    string
	At      time.Time
}

// EventName implements DomainEvent.
func (VenueRegistered) EventName() string { return "venue.VenueRegistered" }

// OccurredAt implements DomainEvent.
func (e VenueRegistered) OccurredAt() time.Time { return e.At }

// SectionAdded records that a section was added to a draft venue's layout.
type SectionAdded struct {
	VenueID VenueID
	Section Section
	At      time.Time
}

// EventName implements DomainEvent.
func (SectionAdded) EventName() string { return "venue.SectionAdded" }

// OccurredAt implements DomainEvent.
func (e SectionAdded) OccurredAt() time.Time { return e.At }

// VenueActivated records that the venue opened for shows. It carries the
// full layout, because downstream contexts (Show, then Ticketing) need it.
type VenueActivated struct {
	VenueID  VenueID
	Name     string
	Country  string // where it is: VAT depends on it (9.8)
	Sections []Section
	At       time.Time
}

// EventName implements DomainEvent.
func (VenueActivated) EventName() string { return "venue.VenueActivated" }

// OccurredAt implements DomainEvent.
func (e VenueActivated) OccurredAt() time.Time { return e.At }

// VenueRetired records that the venue closed for good.
type VenueRetired struct {
	VenueID VenueID
	At      time.Time
}

// EventName implements DomainEvent.
func (VenueRetired) EventName() string { return "venue.VenueRetired" }

// OccurredAt implements DomainEvent.
func (e VenueRetired) OccurredAt() time.Time { return e.At }
