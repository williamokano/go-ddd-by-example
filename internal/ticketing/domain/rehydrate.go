package domain

import "time"

// InventoryState is everything a repository stores about an inventory.
type InventoryState struct {
	ShowID   ShowID
	StartsAt time.Time
	Seats    []SeatView
	Holds    []HoldState
	Closed   bool
	SoldOut  bool
	Version  int
}

// HoldState is a stored hold.
type HoldState struct {
	ID        HoldID
	Customer  CustomerID
	Seats     []SeatRef
	ExpiresAt time.Time
}

// StateOf reads an inventory out for storage.
func StateOf(inv *ShowInventory) InventoryState {
	state := InventoryState{
		ShowID: inv.showID, StartsAt: inv.startsAt, Seats: inv.Seats(),
		Closed: inv.closed, SoldOut: inv.soldOut, Version: inv.version,
	}
	for _, h := range inv.Holds() {
		state.Holds = append(state.Holds, HoldState{ID: h.id, Customer: h.customer, Seats: h.Seats(), ExpiresAt: h.expiresAt})
	}
	return state
}

// RehydrateInventory rebuilds an inventory from storage: no rules, no events.
// Only driven adapters may call it.
func RehydrateInventory(s InventoryState) *ShowInventory {
	inv := &ShowInventory{
		showID: s.ShowID, startsAt: s.StartsAt, seats: make(map[SeatRef]*seat, len(s.Seats)),
		holds: make(map[HoldID]Hold, len(s.Holds)), closed: s.Closed, soldOut: s.SoldOut, version: s.Version,
	}
	for _, v := range s.Seats {
		inv.seats[v.Ref] = &seat{price: v.Price, state: v.State, holdID: v.HoldID, orderID: v.OrderID}
		inv.order = append(inv.order, v.Ref)
	}
	for _, h := range s.Holds {
		inv.holds[h.ID] = Hold{id: h.ID, customer: h.Customer, seats: h.Seats, expiresAt: h.ExpiresAt}
	}
	return inv
}
