package domain

import "time"

// DomainEvent is something that happened in the Venue Management domain that
// domain experts care about. Events are named in the past tense and immutable.
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// Events records the domain events of an aggregate until they are pulled.
// The aggregate records; it never publishes (it knows nothing about Kafka).
type Events struct{ pending []DomainEvent }

// Record appends an event.
func (e *Events) Record(ev DomainEvent) { e.pending = append(e.pending, ev) }

// PullEvents returns the recorded events in order and forgets them.
func (e *Events) PullEvents() []DomainEvent {
	out := e.pending
	e.pending = nil
	return out
}

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
