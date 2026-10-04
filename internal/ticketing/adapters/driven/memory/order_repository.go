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

// OrderRepository is an in-memory application.OrderRepository.
type OrderRepository struct {
	mu        sync.Mutex
	orders    map[domain.OrderID]domain.OrderState
	published []sharedkernel.DomainEvent
}

// NewOrderRepository returns an empty repository.
func NewOrderRepository() *OrderRepository {
	return &OrderRepository{orders: map[domain.OrderID]domain.OrderState{}}
}

// Get implements application.OrderRepository.
func (r *OrderRepository) Get(_ context.Context, id domain.OrderID) (*domain.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.orders[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", application.ErrOrderNotFound, id)
	}
	return domain.RehydrateOrder(s), nil
}

// Save implements application.OrderRepository.
func (r *OrderRepository) Save(_ context.Context, o *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.orders[o.ID()].Version != o.Version() {
		return fmt.Errorf("%w: order %s", application.ErrConcurrentModification, o.ID())
	}
	s := domain.OrderStateOf(o)
	s.Version++
	r.orders[o.ID()] = s
	r.published = append(r.published, o.PullEvents()...)
	return nil
}

// ListPaidForShow implements application.OrderRepository.
func (r *OrderRepository) ListPaidForShow(_ context.Context, show domain.ShowID) ([]*domain.Order, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.Order
	for _, s := range r.orders {
		if s.ShowID == show && (s.Status == domain.Paid || s.Status == domain.Fulfilled) {
			out = append(out, domain.RehydrateOrder(s))
		}
	}
	return out, nil
}

// Published returns every event drained by Save.
func (r *OrderRepository) Published() []sharedkernel.DomainEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.published)
}
