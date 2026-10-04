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

// ticketNamespace scopes the name-based (v5) ticket IDs.
var ticketNamespace = uuid.MustParse("6f3a0c8e-2b1d-5e7f-9a4c-1d2e3f405162")

// TicketIDFor derives a ticket ID from order + seat (UUIDv5): deterministic.
func (TicketingIDs) TicketIDFor(order domain.OrderID, seat domain.SeatRef) domain.TicketID {
	return domain.NewTicketID(uuid.NewSHA1(ticketNamespace, []byte(order.String()+"/"+seat.String())))
}
