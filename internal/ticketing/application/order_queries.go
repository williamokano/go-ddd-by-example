package application

import (
	"context"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// OrderView is an order as its customer sees it.
type OrderView struct {
	ID       string
	ShowID   string
	Status   string
	Amount   int64
	Currency string
	Tickets  []TicketView
}

// TicketView is one ticket of an order.
type TicketView struct {
	Seat   string
	Code   string
	Status string
}

// OrderQueries reads orders and their tickets.
type OrderQueries struct {
	orders  OrderRepository
	tickets TicketRepository
}

// NewOrderQueries reads through the repositories: an order and its few
// tickets are small, so there's no need for a separate read model.
func NewOrderQueries(orders OrderRepository, tickets TicketRepository) *OrderQueries {
	return &OrderQueries{orders: orders, tickets: tickets}
}

// Get returns the order's view, or ErrOrderNotFound.
func (q *OrderQueries) Get(ctx context.Context, id domain.OrderID) (OrderView, error) {
	o, err := q.orders.Get(ctx, id)
	if err != nil {
		return OrderView{}, fmt.Errorf("get order: %w", err)
	}
	tickets, err := q.tickets.ListByOrder(ctx, id)
	if err != nil {
		return OrderView{}, fmt.Errorf("get order tickets: %w", err)
	}
	v := OrderView{
		ID: o.ID().String(), ShowID: o.ShowID().String(), Status: o.Status().String(),
		Amount: o.Total().Amount(), Currency: o.Total().Currency().String(), Tickets: []TicketView{},
	}
	for _, t := range tickets {
		v.Tickets = append(v.Tickets, TicketView{Seat: t.Seat().String(), Code: t.Code().String(), Status: t.Status().String()})
	}
	return v, nil
}
