// Package ids implements Show's IDGenerator port on a platform UUID source.
package ids

import (
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// UUIDSource hands out UUIDs (idgen.UUIDv7, or idgen.Sequence in tests).
type UUIDSource interface {
	New() uuid.UUID
}

// ShowIDs implements application.IDGenerator.
type ShowIDs struct{ source UUIDSource }

// NewShowIDs returns an IDGenerator backed by source.
func NewShowIDs(source UUIDSource) ShowIDs { return ShowIDs{source: source} }

// NewShowID returns a new show identity.
func (g ShowIDs) NewShowID() domain.ShowID { return domain.NewShowID(g.source.New()) }
