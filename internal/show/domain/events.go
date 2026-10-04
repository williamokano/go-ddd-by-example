package domain

import "time"

// DomainEvent is something that happened in Show Scheduling. (A copy of
// Venue's building block: Part 7 promotes it to the shared kernel, ADR-009.)
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// Events records an aggregate's domain events until they are pulled.
type Events struct{ pending []DomainEvent }

// Record appends an event.
func (e *Events) Record(ev DomainEvent) { e.pending = append(e.pending, ev) }

// PullEvents returns the recorded events in order and forgets them.
func (e *Events) PullEvents() []DomainEvent {
	out := e.pending
	e.pending = nil
	return out
}

// ShowDrafted records that a promoter drafted a show at a venue.
type ShowDrafted struct {
	ShowID     ShowID
	VenueID    VenueID
	PromoterID PromoterID
	Title      string
	Schedule   Schedule
	At         time.Time
}

// EventName implements DomainEvent.
func (ShowDrafted) EventName() string { return "show.ShowDrafted" }

// OccurredAt implements DomainEvent.
func (e ShowDrafted) OccurredAt() time.Time { return e.At }
