// Package ids implements the Venue context's IDGenerator port on top of a
// platform UUID source.
package ids

import (
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// UUIDSource is anything that hands out UUIDs: idgen.UUIDv7 in production,
// idgen.Sequence in tests.
type UUIDSource interface {
	New() uuid.UUID
}

// VenueIDs implements application.IDGenerator.
type VenueIDs struct{ source UUIDSource }

// NewVenueIDs returns an IDGenerator backed by source.
func NewVenueIDs(source UUIDSource) VenueIDs { return VenueIDs{source: source} }

// NewVenueID returns a new venue identity.
func (g VenueIDs) NewVenueID() domain.VenueID { return domain.NewVenueID(g.source.New()) }
