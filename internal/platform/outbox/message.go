// Package outbox implements the transactional outbox (ADR-004): integration
// events are written to a <schema>.outbox table in the same transaction as
// the aggregate, and a relay publishes them afterwards. It knows nothing about
// any context: each context's repository translates its own events.
package outbox

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Message is one integration event waiting in an outbox.
type Message struct {
	EventID    uuid.UUID       // unique per event: the consumers' dedup key
	Topic      string          // one topic per producing context
	Key        string          // the aggregate ID: per-aggregate ordering
	Type       string          // e.g. "venue.activated.v1"
	Payload    json.RawMessage // the contracts struct, as JSON
	OccurredAt time.Time
}
