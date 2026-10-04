// Package ids implements Ticketing's IDGenerator port on a platform UUID source.
package ids

import (
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// UUIDSource hands out UUIDs (idgen.UUIDv7, or idgen.Sequence in tests).
type UUIDSource interface {
	New() uuid.UUID
}

// TicketingIDs implements application.IDGenerator.
type TicketingIDs struct{ source UUIDSource }

// New returns an IDGenerator backed by source.
func New(source UUIDSource) TicketingIDs { return TicketingIDs{source: source} }

// NewHoldID returns a new hold identity.
func (g TicketingIDs) NewHoldID() domain.HoldID { return domain.NewHoldID(g.source.New()) }

// NewOrderID returns a new order identity.
func (g TicketingIDs) NewOrderID() domain.OrderID { return domain.NewOrderID(g.source.New()) }
