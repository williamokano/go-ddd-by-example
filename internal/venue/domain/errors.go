package domain

import "errors"

// The rulebook of the Venue Management context: every business rule
// violation the domain can report. Callers check them with errors.Is.
var (
	ErrInvalidVenueID = errors.New("venue: invalid venue id")
	ErrInvalidAddress = errors.New("venue: invalid address") // VEN-1
)
