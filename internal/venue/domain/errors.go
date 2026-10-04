package domain

import "errors"

// The rulebook of the Venue Management context: every business rule
// violation the domain can report. Callers check them with errors.Is.
var (
	ErrInvalidVenueID     = errors.New("venue: invalid venue id")
	ErrInvalidAddress     = errors.New("venue: invalid address")      // VEN-1
	ErrInvalidSectionCode = errors.New("venue: invalid section code") // VEN-2
	ErrInvalidSection     = errors.New("venue: invalid section")      // VEN-3
	ErrInvalidRow         = errors.New("venue: invalid row")          // VEN-3
)
