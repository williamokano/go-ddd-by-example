// Package memory holds in-memory driven adapters: real, working
// implementations of the Venue ports, used by fast tests and for running the
// app without Postgres.
package memory

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// VenueRepository is an in-memory application.VenueRepository.
//
// It stores a copy of each venue's state, never the *domain.Venue pointer:
// with the pointer, a use case that forgets to call Save would still pass its
// tests, because the stored venue would see the mutation.
type VenueRepository struct {
	mu     sync.Mutex
	venues    map[domain.VenueID]domain.VenueState
	published []domain.DomainEvent
}

// NewVenueRepository returns an empty repository.
func NewVenueRepository() *VenueRepository {
	return &VenueRepository{venues: make(map[domain.VenueID]domain.VenueState)}
}

// Get implements application.VenueRepository.
func (r *VenueRepository) Get(_ context.Context, id domain.VenueID) (*domain.Venue, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.venues[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", application.ErrVenueNotFound, id)
	}
	return domain.RehydrateVenue(state), nil
}

// Save implements application.VenueRepository. Like the Postgres adapter, it
// accepts the save only if the stored version is the one the venue was loaded
// at (0 for a new venue), then stores the next version (ADR-011).
func (r *VenueRepository) Save(_ context.Context, v *domain.Venue) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.venues[v.ID()].Version != v.Version() {
		return fmt.Errorf("%w: venue %s", application.ErrConcurrentModification, v.ID())
	}
	state := stateOf(v)
	state.Version++
	r.venues[v.ID()] = state
	r.published = append(r.published, v.PullEvents()...)
	return nil
}

// Published returns every event drained by Save, in order. It stands in for
// the outbox, so tests can check which facts a use case produced.
func (r *VenueRepository) Published() []domain.DomainEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.published)
}

// stateOf reads the venue back through its getters.
func stateOf(v *domain.Venue) domain.VenueState {
	return domain.VenueState{
		ID:       v.ID(),
		Name:     v.Name(),
		Address:  v.Address(),
		Status:   v.Status(),
		Sections: v.Sections(),
		Version:  v.Version(),
	}
}
