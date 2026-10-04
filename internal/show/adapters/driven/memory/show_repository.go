// Package memory holds in-memory driven adapters for Show: used by fast tests.
package memory

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// ShowRepository is an in-memory application.ShowRepository. It stores copies
// of the state, never the pointer (see Venue's fake for why).
type ShowRepository struct {
	mu        sync.Mutex
	shows     map[domain.ShowID]domain.ShowState
	published []sharedkernel.DomainEvent
}

// NewShowRepository returns an empty repository.
func NewShowRepository() *ShowRepository {
	return &ShowRepository{shows: make(map[domain.ShowID]domain.ShowState)}
}

// Get implements application.ShowRepository.
func (r *ShowRepository) Get(_ context.Context, id domain.ShowID) (*domain.Show, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.shows[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", application.ErrShowNotFound, id)
	}
	return domain.RehydrateShow(state), nil
}

// Save implements application.ShowRepository.
func (r *ShowRepository) Save(_ context.Context, s *domain.Show) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.shows[s.ID()].Version != s.Version() {
		return fmt.Errorf("%w: show %s", application.ErrConcurrentModification, s.ID())
	}
	state := stateOf(s)
	state.Version++
	r.shows[s.ID()] = state
	r.published = append(r.published, s.PullEvents()...)
	return nil
}

// ListOpenAtVenue implements application.ShowRepository.
func (r *ShowRepository) ListOpenAtVenue(_ context.Context, venueID domain.VenueID) ([]*domain.Show, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.Show
	for _, state := range r.shows {
		if state.VenueID == venueID && !state.Status.IsTerminal() {
			out = append(out, domain.RehydrateShow(state))
		}
	}
	return out, nil
}

// Published returns every event drained by Save, in order.
func (r *ShowRepository) Published() []sharedkernel.DomainEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.published)
}

func stateOf(s *domain.Show) domain.ShowState {
	return domain.ShowState{
		ID: s.ID(), VenueID: s.VenueID(), PromoterID: s.PromoterID(), Title: s.Title(),
		Schedule: s.Schedule(), Prices: s.Prices(), Status: s.Status(),
		CancellationReason: s.CancellationReason(), Version: s.Version(),
	}
}
