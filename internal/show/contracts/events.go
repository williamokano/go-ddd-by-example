// Package contracts is Show's Published Language: show.published.v1 and
// show.cancelled.v1. Standard library only; never the domain.
package contracts

import "time"

// Topic is where Show publishes, keyed by show ID.
const Topic = "show.events"

// Event types.
const (
	// TypeShowPublishedV1 is retired (9.3): never published again, but old
	// records stay in the topic until retention drops them, so consumers
	// still recognise it, to skip it.
	TypeShowPublishedV1 = "show.published.v1"
	TypeShowPublishedV2 = "show.published.v2"
	TypeShowCancelledV1 = "show.cancelled.v1"
)

// Section kinds.
const (
	KindSeated = "seated"
	KindGA     = "ga"
)

// PriceV1 is an amount in minor units (cents) of a currency.
type PriceV1 struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// ShowPublishedV2 is ShowPublishedV1 restructured (9.3): a seated section
// lists every seat instead of rows of counts, so each seat can carry its own
// flags. Show publishes both versions until every consumer reads v2.
type ShowPublishedV2 struct {
	ShowID       string      `json:"show_id"`
	VenueID      string      `json:"venue_id"`
	VenueCountry string      `json:"venue_country,omitempty"` // added in 9.8 (additive): VAT depends on it
	Title        string      `json:"title"`
	DoorsOpen    time.Time   `json:"doors_open"`
	StartsAt     time.Time   `json:"starts_at"`
	EndsAt       time.Time   `json:"ends_at"`
	Sections     []SectionV2 `json:"sections"`
	PublishedAt  time.Time   `json:"published_at"`
}

// SectionV2 is one priced section: its seats when seated, a capacity when GA.
type SectionV2 struct {
	Code     string   `json:"code"`
	Kind     string   `json:"kind"`
	Seats    []SeatV2 `json:"seats,omitempty"`
	Capacity int      `json:"capacity,omitempty"`
	Price    PriceV1  `json:"price"`
}

// SeatV2 is one seat of a seated section.
type SeatV2 struct {
	Row        string `json:"row"`
	Number     int    `json:"number"`
	Accessible bool   `json:"accessible,omitempty"`
}

// ShowCancelledV1 announces a show was cancelled: Ticketing closes the
// inventory, voids tickets and refunds orders (TKT-11).
type ShowCancelledV1 struct {
	ShowID      string    `json:"show_id"`
	VenueID     string    `json:"venue_id"`
	Reason      string    `json:"reason"`
	CancelledAt time.Time `json:"cancelled_at"`
}
