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
	inventories map[domain.ShowID]domain.InventoryState
	published   []sharedkernel.DomainEvent
}

// NewInventoryRepository returns an empty repository.
func NewInventoryRepository() *InventoryRepository {
	return &InventoryRepository{inventories: make(map[domain.ShowID]domain.InventoryState)}
}

// Get implements application.InventoryRepository.
func (r *InventoryRepository) Get(_ context.Context, id domain.ShowID) (*domain.ShowInventory, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.inventories[id]
	if !ok {
		return nil, fmt.Errorf("%w: show %s", application.ErrInventoryNotFound, id)
	}
	return domain.RehydrateInventory(clone(state)), nil
}

// GetByHold implements application.InventoryRepository.
func (r *InventoryRepository) GetByHold(_ context.Context, id domain.HoldID) (*domain.ShowInventory, error) {
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
func (r *InventoryRepository) Save(_ context.Context, inv *domain.ShowInventory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, exists := r.inventories[inv.ShowID()]
	if exists != (inv.Version() > 0) || stored.Version != inv.Version() {
		return fmt.Errorf("%w: inventory %s", application.ErrConcurrentModification, inv.ShowID())
	}
	state := domain.StateOf(inv)
	state.Version++
	r.inventories[inv.ShowID()] = state
	r.published = append(r.published, inv.PullEvents()...)
	return nil
}

// Published returns every event drained by Save, in order.
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
