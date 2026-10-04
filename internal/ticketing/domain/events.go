package domain

import (
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
)

// InventoryOpened records that a published show's inventory opened (TKT-1).
type InventoryOpened struct {
	ShowID ShowID
	Seats  int
	At     time.Time
}

// SeatsHeld records a new hold (TKT-2).
type SeatsHeld struct {
	ShowID     ShowID
	HoldID     HoldID
	CustomerID CustomerID
	Seats      []SeatRef
	ExpiresAt  time.Time
	At         time.Time
}

// HoldExpired records that a hold lapsed and its seats are available (TKT-4).
type HoldExpired struct {
	ShowID ShowID
	HoldID HoldID
	Seats  []SeatRef
	At     time.Time
}

// EventName implements DomainEvent.
func (InventoryOpened) EventName() string { return "ticketing.InventoryOpened" }

// OccurredAt implements DomainEvent.
func (e InventoryOpened) OccurredAt() time.Time { return e.At }

// EventName implements DomainEvent.
func (SeatsHeld) EventName() string { return "ticketing.SeatsHeld" }

// OccurredAt implements DomainEvent.
func (e SeatsHeld) OccurredAt() time.Time { return e.At }

// EventName implements DomainEvent.
func (HoldExpired) EventName() string { return "ticketing.HoldExpired" }

// OccurredAt implements DomainEvent.
func (e HoldExpired) OccurredAt() time.Time { return e.At }

// HoldReleased records that a customer gave their held seats back.
type HoldReleased struct {
	ShowID ShowID
	HoldID HoldID
	Seats  []SeatRef
	At     time.Time
}

// EventName implements DomainEvent.
func (HoldReleased) EventName() string { return "ticketing.HoldReleased" }

// OccurredAt implements DomainEvent.
func (e HoldReleased) OccurredAt() time.Time { return e.At }

// SeatsSold records that a paid order's held seats are sold (TKT-8).
type SeatsSold struct {
	ShowID  ShowID
	HoldID  HoldID
	OrderID OrderID
	Seats   []SeatRef
	At      time.Time
}

// EventName implements DomainEvent.
func (SeatsSold) EventName() string { return "ticketing.SeatsSold" }

// OccurredAt implements DomainEvent.
func (e SeatsSold) OccurredAt() time.Time { return e.At }

// InventorySoldOut records that every seat is sold (TKT-10).
type InventorySoldOut struct {
	ShowID ShowID
	At     time.Time
}

// EventName implements DomainEvent.
func (InventorySoldOut) EventName() string { return "ticketing.InventorySoldOut" }

// OccurredAt implements DomainEvent.
func (e InventorySoldOut) OccurredAt() time.Time { return e.At }

// InventoryClosed records that sales stopped for good (TKT-11).
type InventoryClosed struct {
	ShowID        ShowID
	ReleasedSeats []SeatRef // the seats of the holds it released
	At            time.Time
}

// EventName implements DomainEvent.
func (InventoryClosed) EventName() string { return "ticketing.InventoryClosed" }

// OccurredAt implements DomainEvent.
func (e InventoryClosed) OccurredAt() time.Time { return e.At }

// OrderPlaced records that a customer checked out a hold (TKT-6).
type OrderPlaced struct {
	OrderID    OrderID
	ShowID     ShowID
	HoldID     HoldID
	CustomerID CustomerID
	Total      sharedkernel.Money
	At         time.Time
}

// EventName implements DomainEvent.
func (OrderPlaced) EventName() string { return "ticketing.OrderPlaced" }

// OccurredAt implements DomainEvent.
func (e OrderPlaced) OccurredAt() time.Time { return e.At }

// OrderPaid records a successful charge: the saga confirms the hold next.
type OrderPaid struct {
	OrderID OrderID
	ShowID  ShowID
	HoldID  HoldID
	At      time.Time
}

// EventName implements DomainEvent.
func (OrderPaid) EventName() string { return "ticketing.OrderPaid" }

// OccurredAt implements DomainEvent.
func (e OrderPaid) OccurredAt() time.Time { return e.At }

// OrderPaymentFailed records a declined charge.
type OrderPaymentFailed struct {
	OrderID OrderID
	Reason  string
	At      time.Time
}

// EventName implements DomainEvent.
func (OrderPaymentFailed) EventName() string { return "ticketing.OrderPaymentFailed" }

// OccurredAt implements DomainEvent.
func (e OrderPaymentFailed) OccurredAt() time.Time { return e.At }

// OrderFulfilled records that the order's tickets were issued.
type OrderFulfilled struct {
	OrderID OrderID
	At      time.Time
}

// EventName implements DomainEvent.
func (OrderFulfilled) EventName() string { return "ticketing.OrderFulfilled" }

// OccurredAt implements DomainEvent.
func (e OrderFulfilled) OccurredAt() time.Time { return e.At }

// OrderRefunded records that the money went back; it carries where to tell the customer.
type OrderRefunded struct {
	OrderID      OrderID
	ShowID       ShowID
	CustomerID   CustomerID
	ContactEmail ContactEmail
	Total        sharedkernel.Money
	At           time.Time
}

// EventName implements DomainEvent.
func (OrderRefunded) EventName() string { return "ticketing.OrderRefunded" }

// OccurredAt implements DomainEvent.
func (e OrderRefunded) OccurredAt() time.Time { return e.At }
