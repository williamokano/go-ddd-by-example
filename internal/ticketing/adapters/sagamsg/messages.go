// Package sagamsg holds the checkout saga's private messages on
// ticketing.internal. They are not a Published Language: no other context
// subscribes, and they may change freely with Ticketing. Both the outbox
// translator (driven) and the saga consumer (driving) use them.
package sagamsg

// Topic is the saga's private topic, keyed by order ID.
const Topic = "ticketing.internal"

// Message types.
const (
	TypeOrderPaid              = "ticketing.order_paid"
	TypeSeatsSold              = "ticketing.seats_sold"
	TypeHoldConfirmationFailed = "ticketing.hold_confirmation_failed"
	TypeInventoryClosed        = "ticketing.inventory_closed"
	TypeSectionSoldOut         = "ticketing.section_sold_out"
)

// OrderPaid starts ConfirmHold.
type OrderPaid struct {
	OrderID string `json:"order_id"`
	ShowID  string `json:"show_id"`
	Section string `json:"section"`
	HoldID  string `json:"hold_id"`
}

// SectionSoldOut starts the show-wide sold-out check (TKT-10, ADR-013).
type SectionSoldOut struct {
	ShowID  string `json:"show_id"`
	Section string `json:"section"`
}

// SeatsSold starts IssueTickets.
type SeatsSold struct {
	OrderID string   `json:"order_id"`
	ShowID  string   `json:"show_id"`
	Seats   []string `json:"seats"`
}

// HoldConfirmationFailed starts RefundOrder: the compensation.
type HoldConfirmationFailed struct {
	OrderID string `json:"order_id"`
	ShowID  string `json:"show_id"`
	Reason  string `json:"reason"`
}

// InventoryClosed starts voiding the show's tickets and refunding its orders
// (TKT-11).
type InventoryClosed struct {
	ShowID string `json:"show_id"`
}
