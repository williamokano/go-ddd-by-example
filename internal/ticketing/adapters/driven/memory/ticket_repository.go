package memory

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// TicketRepository is an in-memory application.TicketRepository.
type TicketRepository struct {
	mu      sync.Mutex
	tickets map[domain.TicketID]domain.TicketState
}

// NewTicketRepository returns an empty repository.
func NewTicketRepository() *TicketRepository {
	return &TicketRepository{tickets: map[domain.TicketID]domain.TicketState{}}
}

// Save implements application.TicketRepository.
func (r *TicketRepository) Save(_ context.Context, t *domain.Ticket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored, exists := r.tickets[t.ID()]
	if t.Version() == 0 && exists {
		t.PullEvents()
		return nil // already issued: a redelivery
	}
	if stored.Version != t.Version() {
		return fmt.Errorf("%w: ticket %s", application.ErrConcurrentModification, t.ID())
	}
	s := domain.TicketStateOf(t)
	s.Version++
	r.tickets[t.ID()] = s
	t.PullEvents()
	return nil
}

// GetByCode implements application.TicketRepository.
func (r *TicketRepository) GetByCode(_ context.Context, code domain.TicketCode) (*domain.Ticket, error) {
	found := r.list(func(s domain.TicketState) bool { return s.Code == code })
	if len(found) == 0 {
		return nil, fmt.Errorf("%w: %s", application.ErrTicketNotFound, code)
	}
	return found[0], nil
}

// ListByOrder implements application.TicketRepository.
func (r *TicketRepository) ListByOrder(_ context.Context, order domain.OrderID) ([]*domain.Ticket, error) {
	return r.list(func(s domain.TicketState) bool { return s.OrderID == order }), nil
}

// ListByShow implements application.TicketRepository.
func (r *TicketRepository) ListByShow(_ context.Context, show domain.ShowID) ([]*domain.Ticket, error) {
	return r.list(func(s domain.TicketState) bool { return s.ShowID == show }), nil
}

func (r *TicketRepository) list(keep func(domain.TicketState) bool) []*domain.Ticket {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.Ticket
	for _, s := range r.tickets {
		if keep(s) {
			out = append(out, domain.RehydrateTicket(s))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Seat().String() < out[j].Seat().String() })
	return out
}
