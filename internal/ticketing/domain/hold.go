package domain

import (
	"slices"
	"time"
)

// Hold is a temporary, exclusive claim by a customer on some seats (an entity
// inside the SectionInventory, identified by its HoldID).
type Hold struct {
	id        HoldID
	customer  CustomerID
	seats     []SeatRef
	expiresAt time.Time
}

// ID returns the hold's identity.
func (h Hold) ID() HoldID { return h.id }

// Customer returns who holds the seats.
func (h Hold) Customer() CustomerID { return h.customer }

// Seats returns a copy of the held seats.
func (h Hold) Seats() []SeatRef { return slices.Clone(h.seats) }

// ExpiresAt returns when the hold lapses (TKT-4).
func (h Hold) ExpiresAt() time.Time { return h.expiresAt }

// IsExpired reports whether the hold has lapsed at now.
func (h Hold) IsExpired(now time.Time) bool { return !now.Before(h.expiresAt) }
