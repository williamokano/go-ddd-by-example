// Package idgen generates identities. Contexts wrap it in their own
// IDGenerator adapters, which turn a UUID into a typed ID (VenueID, ShowID…):
// platform code must not import any context.
package idgen

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// UUIDv7 generates time-ordered UUIDs (ADR-008): they index well in Postgres.
type UUIDv7 struct{}

// New returns a new UUIDv7.
func (UUIDv7) New() uuid.UUID { return uuid.Must(uuid.NewV7()) }

// Sequence generates predictable UUIDs (…0001, …0002, …) so tests can know
// the next ID in advance.
type Sequence struct {
	mu   sync.Mutex
	next uint64
}

// NewSequence returns a sequence starting at 1.
func NewSequence() *Sequence { return &Sequence{next: 1} }

// New returns the next UUID of the sequence.
func (s *Sequence) New() uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := uuid.MustParse(fmt.Sprintf("00000000-0000-7000-8000-%012x", s.next))
	s.next++
	return id
}
