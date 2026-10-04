// Package contracts is Ticketing's Published Language on ticketing.events.
// Standard library only; never the domain.
package contracts

import "time"

// Topic is where Ticketing publishes: sold-out facts keyed by show, order
// facts keyed by order.
const Topic = "ticketing.events"

// Event types.
const (
	TypeInventorySoldOutV1 = "ticketing.inventory_sold_out.v1"
	TypeTicketsIssuedV1    = "ticketing.tickets_issued.v1"
	TypeOrderRefundedV1    = "ticketing.order_refunded.v1"
)

// InventorySoldOutV1 announces every seat of a show is sold (TKT-10). Show
// conforms: the show becomes SoldOut (SHW-8).
type InventorySoldOutV1 struct {
	ShowID    string    `json:"show_id"`
	SoldOutAt time.Time `json:"sold_out_at"`
}

// TicketsIssuedV1 announces an order's tickets, and where to send them.
type TicketsIssuedV1 struct {
	OrderID      string     `json:"order_id"`
	ShowID       string     `json:"show_id"`
	CustomerID   string     `json:"customer_id"`
	ContactEmail string     `json:"contact_email"`
	Tickets      []TicketV1 `json:"tickets"`
	IssuedAt     time.Time  `json:"issued_at"`
}

// TicketV1 is one issued ticket.
type TicketV1 struct {
	Seat string `json:"seat"`
	Code string `json:"code"`
}

// OrderRefundedV1 announces an order's money went back, and where to say so.
type OrderRefundedV1 struct {
	OrderID      string    `json:"order_id"`
	ShowID       string    `json:"show_id"`
	CustomerID   string    `json:"customer_id"`
	ContactEmail string    `json:"contact_email"`
	Amount       int64     `json:"amount"`
	Currency     string    `json:"currency"`
	RefundedAt   time.Time `json:"refunded_at"`
}
