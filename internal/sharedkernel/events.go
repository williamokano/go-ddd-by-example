package sharedkernel

import "time"

// DomainEvent is something that happened in a context's domain that domain
// experts care about. Events are named in the past tense and immutable.
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// Events records an aggregate's domain events until they are pulled. The
// aggregate records; it never publishes. Keep it a named, unexported field of
// the root: embedding would promote Record and let callers fake history.
type Events struct{ pending []DomainEvent }

// Record appends an event.
func (e *Events) Record(ev DomainEvent) { e.pending = append(e.pending, ev) }

// PullEvents returns the recorded events in order and forgets them.
func (e *Events) PullEvents() []DomainEvent {
	out := e.pending
	e.pending = nil
	return out
}
