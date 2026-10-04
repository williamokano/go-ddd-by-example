// Package contracts is Show's Published Language: show.published.v1 and
// show.cancelled.v1. Standard library only; never the domain.
package contracts

import "time"

// Topic is where Show publishes, keyed by show ID.
const Topic = "show.events"

// Event types.
const (
	TypeShowPublishedV1 = "show.published.v1"
	TypeShowCancelledV1 = "show.cancelled.v1"
)

// Section kinds.
const (
	KindSeated = "seated"
	KindGA     = "ga"
)

// ShowPublishedV1 announces a show went on sale. It carries a snapshot of the
// layout and prices, so Ticketing can open the inventory without asking
// anyone; later layout changes never affect an already published show.
type ShowPublishedV1 struct {
	ShowID      string      `json:"show_id"`
	VenueID     string      `json:"venue_id"`
	Title       string      `json:"title"`
	DoorsOpen   time.Time   `json:"doors_open"`
	StartsAt    time.Time   `json:"starts_at"`
	EndsAt      time.Time   `json:"ends_at"`
	Sections    []SectionV1 `json:"sections"`
	PublishedAt time.Time   `json:"published_at"`
}

// SectionV1 is one priced section: rows when seated, a capacity when GA.
type SectionV1 struct {
	Code     string  `json:"code"`
	Kind     string  `json:"kind"`
	Rows     []RowV1 `json:"rows,omitempty"`
	Capacity int     `json:"capacity,omitempty"`
	Price    PriceV1 `json:"price"`
}

// RowV1 is one row; its seats are numbered 1..Seats.
type RowV1 struct {
	Label string `json:"label"`
	Seats int    `json:"seats"`
}

// PriceV1 is an amount in minor units (cents) of a currency.
type PriceV1 struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// ShowCancelledV1 announces a show was cancelled: Ticketing closes the
// inventory, voids tickets and refunds orders (TKT-11).
type ShowCancelledV1 struct {
	ShowID      string    `json:"show_id"`
	VenueID     string    `json:"venue_id"`
	Reason      string    `json:"reason"`
	CancelledAt time.Time `json:"cancelled_at"`
}
