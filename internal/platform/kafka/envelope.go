// Package kafka is the Kafka plumbing (franz-go): a producer that implements
// the outbox relay's Publisher, and a consumer-group runner that hands each
// message's envelope to a context's handler.
package kafka

import (
	"encoding/json"
	"time"
)

// Envelope is the value of every message on our topics. Its fields also go
// into Kafka headers, so consumers can route on event_type without decoding.
type Envelope struct {
	EventID       string          `json:"event_id"` // the dedup key
	EventType     string          `json:"event_type"`
	OccurredAt    time.Time       `json:"occurred_at"`
	AggregateID   string          `json:"aggregate_id"`
	CorrelationID string          `json:"correlation_id,omitempty"` // Part 8
	CausationID   string          `json:"causation_id,omitempty"`   // Part 8
	Payload       json.RawMessage `json:"payload"`                  // a contracts struct
}
