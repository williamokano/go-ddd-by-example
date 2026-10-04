package domain

// Status is where a venue is in its lifecycle: Draft → Active → Retired.
type Status uint8

// The venue lifecycle.
const (
	Draft Status = iota + 1
	Active
	Retired
)
