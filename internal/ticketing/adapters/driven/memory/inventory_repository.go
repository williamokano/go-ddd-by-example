// Package memory holds Ticketing's in-memory driven adapters, for fast tests.
package memory

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// InventoryRepository is an in-memory application.InventoryRepository that
// stores copies of the state, never pointers.
type InventoryRepository struct {
	mu          sync.Mutex
	inventories map[application.SectionKey]domain.InventoryState
	published   []sharedkernel.DomainEvent
}

// NewInventoryRepository returns an empty repository.
func NewInventoryRepository() *InventoryRepository {
	return &InventoryRepository{inventories: make(map[application.SectionKey]domain.InventoryState)}
}

// Get implements application.InventoryRepository.
func (r *InventoryRepository) Get(_ context.Context, id domain.ShowID, section string) (*domain.SectionInventory, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.inventories[application.SectionKey{ShowID: id, Section: section}]
	if !ok {
		return nil, fmt.Errorf("%w: show %s section %s", application.ErrInventoryNotFound, id, section)
	}
	return domain.RehydrateInventory(clone(state)), nil
}

// ListByShow implements application.InventoryRepository.
func (r *InventoryRepository) ListByShow(_ context.Context, id domain.ShowID) ([]*domain.SectionInventory, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.SectionInventory
	for key, state := range r.inventories {
		if key.ShowID == id {
			out = append(out, domain.RehydrateInventory(clone(state)))
		}
	}
	return out, nil
}

// GetByHold implements application.InventoryRepository.
func (r *InventoryRepository) GetByHold(_ context.Context, id domain.HoldID) (*domain.SectionInventory, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, state := range r.inventories {
		for _, h := range state.Holds {
			if h.ID == id {
				return domain.RehydrateInventory(clone(state)), nil
			}
		}
	}
	return nil, fmt.Errorf("%w: %s", application.ErrHoldNotFound, id)
}

// Save implements application.InventoryRepository.
func (r *InventoryRepository) Save(_ context.Context, inv *domain.SectionInventory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := application.SectionKey{ShowID: inv.ShowID(), Section: inv.Section()}
	stored, exists := r.inventories[key]
	if exists != (inv.Version() > 0) || stored.Version != inv.Version() {
		return fmt.Errorf("%w: inventory %s section %s", application.ErrConcurrentModification, inv.ShowID(), inv.Section())
	}
	state := domain.StateOf(inv)
	state.Version++
	r.inventories[key] = state
	r.published = append(r.published, inv.PullEvents()...)
	return nil
}

// Publish implements application.EventPublisher: the fake keeps one stream of
// everything announced, by Save or by a domain service.
func (r *InventoryRepository) Publish(_ context.Context, events ...sharedkernel.DomainEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.published = append(r.published, events...)
	return nil
}

// Published returns every event drained by Save or published, in order.
func (r *InventoryRepository) Published() []sharedkernel.DomainEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.published)
}

func clone(s domain.InventoryState) domain.InventoryState {
	s.Seats = slices.Clone(s.Seats)
	s.Holds = slices.Clone(s.Holds)
	return s
}
