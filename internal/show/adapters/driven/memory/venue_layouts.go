package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// VenueLayouts is an in-memory application.VenueLayouts.
type VenueLayouts struct {
	mu      sync.Mutex
	layouts map[domain.VenueID]domain.VenueLayout
}

// NewVenueLayouts returns an empty projection.
func NewVenueLayouts() *VenueLayouts {
	return &VenueLayouts{layouts: make(map[domain.VenueID]domain.VenueLayout)}
}

// Get implements application.VenueLayouts.
func (l *VenueLayouts) Get(_ context.Context, id domain.VenueID) (domain.VenueLayout, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	layout, ok := l.layouts[id]
	if !ok {
		return domain.VenueLayout{}, fmt.Errorf("%w: %s", application.ErrVenueUnknown, id)
	}
	return layout, nil
}

// Upsert implements application.VenueLayouts.
func (l *VenueLayouts) Upsert(_ context.Context, layout domain.VenueLayout) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.layouts[layout.VenueID] = layout
	return nil
}
