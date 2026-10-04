// Package contracts is the Venue context's Published Language: the public,
// versioned integration events other contexts may decode. It imports only the
// standard library and never the domain, so consumers can depend on it without
// depending on Venue's model. Changes are additive within a version; a
// breaking change is a new version (v2), published side by side.
package contracts

import "time"

// Topic is where Venue publishes, keyed by venue ID (per-venue ordering).
const Topic = "venue.events"

// Event types.
const (
	TypeVenueActivatedV1 = "venue.activated.v1"
	TypeVenueRetiredV1   = "venue.retired.v1"
)

// Section kinds.
const (
	KindSeated = "seated"
	KindGA     = "ga"
)

// VenueActivatedV1: a venue opened for shows. It carries the whole layout, so
// consumers never have to ask Venue anything.
type VenueActivatedV1 struct {
	VenueID     string      `json:"venue_id"`
	Name        string      `json:"name"`
	Sections    []SectionV1 `json:"sections"`
	ActivatedAt time.Time   `json:"activated_at"`
}

// SectionV1 is one section of the layout: rows for a seated section, a
// capacity for a general admission one.
type SectionV1 struct {
	Code     string  `json:"code"`
	Kind     string  `json:"kind"`
	Rows     []RowV1 `json:"rows,omitempty"`
	Capacity int     `json:"capacity,omitempty"`
}

// RowV1 is one row of a seated section; its seats are numbered 1..Seats.
type RowV1 struct {
	Label string `json:"label"`
	Seats int    `json:"seats"`
}

// VenueRetiredV1: a venue closed for good. Shows there must be cancelled (SHW-7).
type VenueRetiredV1 struct {
	VenueID   string    `json:"venue_id"`
	RetiredAt time.Time `json:"retired_at"`
}
