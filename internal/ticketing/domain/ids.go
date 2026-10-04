package domain

import (
	"fmt"

	"github.com/google/uuid"
)

// ShowID identifies a show in Ticketing.
type ShowID struct{ value uuid.UUID }

// NewShowID wraps a UUID.
func NewShowID(u uuid.UUID) ShowID { return ShowID{value: u} }

// ParseShowID parses a ShowID.
func ParseShowID(raw string) (ShowID, error) {
	u, err := parseUUID("show", raw)
	return ShowID{value: u}, err
}

// String returns the canonical textual form.
func (id ShowID) String() string { return id.value.String() }

// UUID returns the underlying UUID (for adapters).
func (id ShowID) UUID() uuid.UUID { return id.value }

// IsZero reports whether the ID is the zero value.
func (id ShowID) IsZero() bool { return id.value == uuid.Nil }

// CustomerID identifies a customer in Ticketing.
type CustomerID struct{ value uuid.UUID }

// NewCustomerID wraps a UUID.
func NewCustomerID(u uuid.UUID) CustomerID { return CustomerID{value: u} }

// ParseCustomerID parses a CustomerID.
func ParseCustomerID(raw string) (CustomerID, error) {
	u, err := parseUUID("customer", raw)
	return CustomerID{value: u}, err
}

// String returns the canonical textual form.
func (id CustomerID) String() string { return id.value.String() }

// UUID returns the underlying UUID (for adapters).
func (id CustomerID) UUID() uuid.UUID { return id.value }

// IsZero reports whether the ID is the zero value.
func (id CustomerID) IsZero() bool { return id.value == uuid.Nil }

// HoldID identifies a hold in Ticketing.
type HoldID struct{ value uuid.UUID }

// NewHoldID wraps a UUID.
func NewHoldID(u uuid.UUID) HoldID { return HoldID{value: u} }

// ParseHoldID parses a HoldID.
func ParseHoldID(raw string) (HoldID, error) {
	u, err := parseUUID("hold", raw)
	return HoldID{value: u}, err
}

// String returns the canonical textual form.
func (id HoldID) String() string { return id.value.String() }

// UUID returns the underlying UUID (for adapters).
func (id HoldID) UUID() uuid.UUID { return id.value }

// IsZero reports whether the ID is the zero value.
func (id HoldID) IsZero() bool { return id.value == uuid.Nil }

// OrderID identifies a order in Ticketing.
type OrderID struct{ value uuid.UUID }

// NewOrderID wraps a UUID.
func NewOrderID(u uuid.UUID) OrderID { return OrderID{value: u} }

// ParseOrderID parses a OrderID.
func ParseOrderID(raw string) (OrderID, error) {
	u, err := parseUUID("order", raw)
	return OrderID{value: u}, err
}

// String returns the canonical textual form.
func (id OrderID) String() string { return id.value.String() }

// UUID returns the underlying UUID (for adapters).
func (id OrderID) UUID() uuid.UUID { return id.value }

// IsZero reports whether the ID is the zero value.
func (id OrderID) IsZero() bool { return id.value == uuid.Nil }

// TicketID identifies a ticket in Ticketing.
type TicketID struct{ value uuid.UUID }

// NewTicketID wraps a UUID.
func NewTicketID(u uuid.UUID) TicketID { return TicketID{value: u} }

// ParseTicketID parses a TicketID.
func ParseTicketID(raw string) (TicketID, error) {
	u, err := parseUUID("ticket", raw)
	return TicketID{value: u}, err
}

// String returns the canonical textual form.
func (id TicketID) String() string { return id.value.String() }

// UUID returns the underlying UUID (for adapters).
func (id TicketID) UUID() uuid.UUID { return id.value }

// IsZero reports whether the ID is the zero value.
func (id TicketID) IsZero() bool { return id.value == uuid.Nil }

func parseUUID(kind, raw string) (uuid.UUID, error) {
	u, err := uuid.Parse(raw)
	if err != nil || u == uuid.Nil {
		return uuid.Nil, fmt.Errorf("%w: %s id %q", ErrInvalidID, kind, raw)
	}
	return u, nil
}
