// Package memory holds in-memory driven adapters: real, working
// implementations of the Venue ports, used by fast tests and for running the
// app without Postgres.
package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// VenueRepository is an in-memory application.VenueRepository.
type VenueRepository struct {
	mu     sync.Mutex
	venues map[domain.VenueID]*domain.Venue
}

// NewVenueRepository returns an empty repository.
func NewVenueRepository() *VenueRepository {
	return &VenueRepository{venues: make(map[domain.VenueID]*domain.Venue)}
}

// Get implements application.VenueRepository.
func (r *VenueRepository) Get(_ context.Context, id domain.VenueID) (*domain.Venue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.venues[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", application.ErrVenueNotFound, id)
	}
	return v, nil
}

// Save implements application.VenueRepository.
func (r *VenueRepository) Save(_ context.Context, v *domain.Venue) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.venues[v.ID()] = v
	return nil
}
