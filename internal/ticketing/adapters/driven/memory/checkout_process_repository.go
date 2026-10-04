package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// CheckoutProcessRepository is an in-memory application.CheckoutProcessRepository.
type CheckoutProcessRepository struct {
	mu        sync.Mutex
	processes map[domain.OrderID]domain.CheckoutProcessState
}

// NewCheckoutProcessRepository returns an empty repository.
func NewCheckoutProcessRepository() *CheckoutProcessRepository {
	return &CheckoutProcessRepository{processes: map[domain.OrderID]domain.CheckoutProcessState{}}
}

// Get implements application.CheckoutProcessRepository.
func (r *CheckoutProcessRepository) Get(_ context.Context, id domain.OrderID) (*domain.CheckoutProcess, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.processes[id]
	if !ok {
		return nil, fmt.Errorf("%w: order %s", application.ErrCheckoutNotFound, id)
	}
	return domain.RehydrateCheckoutProcess(s), nil
}

// Save implements application.CheckoutProcessRepository.
func (r *CheckoutProcessRepository) Save(_ context.Context, p *domain.CheckoutProcess) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, exists := r.processes[p.OrderID()]
	if exists != (p.Version() > 0) || stored.Version != p.Version() {
		return fmt.Errorf("%w: checkout %s", application.ErrConcurrentModification, p.OrderID())
	}
	s := domain.CheckoutProcessStateOf(p)
	s.Version++
	r.processes[p.OrderID()] = s
	p.PullEvents()
	return nil
}
