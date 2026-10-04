package domain

import "github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

// SeatState is where a sellable unit is: Available, Held or Sold.
type SeatState uint8

// Seat states.
const (
	Available SeatState = iota + 1
	Held
	Sold
)

var seatStateNames = map[SeatState]string{Available: "available", Held: "held", Sold: "sold"}

// String returns the stored name of the state.
func (s SeatState) String() string {
	if name, ok := seatStateNames[s]; ok {
		return name
	}
	return "unknown"
}

// SeatView is a read-only copy of one seat of the inventory.
type SeatView struct {
	Ref     SeatRef
	Price   sharedkernel.Money
	State   SeatState
	HoldID  HoldID  // set while Held
	OrderID OrderID // set once Sold

	Accessible bool // step-free access (9.3)
}
