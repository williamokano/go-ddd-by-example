package domain

import (
	"fmt"

	"github.com/google/uuid"
)

// ShowID identifies a show.
type ShowID struct{ value uuid.UUID }

// VenueID is Show's own reference to a venue. It is not venue/domain.VenueID:
// contexts share identifiers' values, never their types.
type VenueID struct{ value uuid.UUID }

// PromoterID identifies the promoter who drafted a show.
type PromoterID struct{ value uuid.UUID }

// NewShowID wraps a generated UUID (ADR-008).
func NewShowID(u uuid.UUID) ShowID { return ShowID{value: u} }

// NewVenueID wraps a UUID.
func NewVenueID(u uuid.UUID) VenueID { return VenueID{value: u} }

// NewPromoterID wraps a UUID.
func NewPromoterID(u uuid.UUID) PromoterID { return PromoterID{value: u} }

// ParseShowID parses a ShowID.
func ParseShowID(raw string) (ShowID, error) {
	u, err := parseUUID("show", raw)
	return ShowID{value: u}, err
}

// ParseVenueID parses a VenueID.
func ParseVenueID(raw string) (VenueID, error) {
	u, err := parseUUID("venue", raw)
	return VenueID{value: u}, err
}

// ParsePromoterID parses a PromoterID.
func ParsePromoterID(raw string) (PromoterID, error) {
	u, err := parseUUID("promoter", raw)
	return PromoterID{value: u}, err
}

func parseUUID(kind, raw string) (uuid.UUID, error) {
	u, err := uuid.Parse(raw)
	if err != nil || u == uuid.Nil {
		return uuid.Nil, fmt.Errorf("%w: %s id %q", ErrInvalidID, kind, raw)
	}
	return u, nil
}

func (id ShowID) String() string     { return id.value.String() }
func (id VenueID) String() string    { return id.value.String() }
func (id PromoterID) String() string { return id.value.String() }

// IsZero reports whether the ID is the zero value.
func (id ShowID) IsZero() bool { return id.value == uuid.Nil }

// IsZero reports whether the ID is the zero value.
func (id VenueID) IsZero() bool { return id.value == uuid.Nil }

// IsZero reports whether the ID is the zero value.
func (id PromoterID) IsZero() bool { return id.value == uuid.Nil }
