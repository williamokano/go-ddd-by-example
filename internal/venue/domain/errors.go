package domain

import "errors"

// The rulebook of the Venue Management context: every business rule
// violation the domain can report, in rule order. Callers check them with
// errors.Is; the HTTP adapter maps them to status codes.
var (
	ErrInvalidVenueID = errors.New("venue: invalid venue id")

	// VEN-1: a venue has a name (≤ 120 chars) and a valid address.
	ErrInvalidVenueName = errors.New("venue: invalid venue name")
	ErrInvalidAddress   = errors.New("venue: invalid address")

	// VEN-2: section codes are well-formed and unique within a venue.
	ErrInvalidSectionCode   = errors.New("venue: invalid section code")
	ErrDuplicateSectionCode = errors.New("venue: duplicate section code")

	// VEN-3: seated sections have unique rows of ≥ 1 seat; GA sections a capacity ≥ 1.
	ErrInvalidSection    = errors.New("venue: invalid section")
	ErrInvalidRow        = errors.New("venue: invalid row")
	ErrDuplicateRowLabel = errors.New("venue: duplicate row label")

	// VEN-4: the layout only changes while the venue is a draft.
	ErrVenueNotDraft = errors.New("venue: layout can only change while draft")

	// VEN-5: only a draft with ≥ 1 section can be activated.
	// VEN-6: only an active venue can be retired; Retired is terminal.
	ErrVenueHasNoSections     = errors.New("venue: cannot activate a venue without sections")
	ErrInvalidVenueTransition = errors.New("venue: invalid lifecycle transition")
)
