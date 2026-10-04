package domain

import "github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

// Section kinds of an inventory layout.
const (
	KindSeated = "seated"
	KindGA     = "ga"
)

// InventoryLayout is the snapshot Show published with the show: what can be
// sold, and at which price. Ticketing never asks Venue or Show anything else.
type InventoryLayout struct {
	Sections []InventorySection
}

// InventorySection is one priced section: rows when seated, a capacity when GA.
type InventorySection struct {
	Code     string
	Kind     string
	Rows     []InventoryRow
	Capacity int
	Price    sharedkernel.Money
}

// InventoryRow is one row of seats, numbered 1..Seats.
type InventoryRow struct {
	Label      string
	Seats      int
	Accessible []int // seat numbers with step-free access
}
