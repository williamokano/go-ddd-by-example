package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// gaRow is the row label of general admission places: "FLOOR/GA/0457".
const gaRow = "GA"

var sectionCode = regexp.MustCompile(`^[A-Z0-9-]{1,10}$`)

// SeatRef identifies a sellable unit of one show: section/row/number for a
// seat ("ORCH/A/12"), section/GA/number for a standing place ("FLOOR/GA/0457").
// Ticketing's Seat is not Venue's: it exists to be held and sold.
type SeatRef struct {
	section string
	row     string
	number  int
}

// NewSeatRef builds a reference from its parts.
func NewSeatRef(section, row string, number int) (SeatRef, error) {
	section, row = strings.ToUpper(strings.TrimSpace(section)), strings.ToUpper(strings.TrimSpace(row))
	if !sectionCode.MatchString(section) || row == "" || strings.Contains(row, "/") || number < 1 {
		return SeatRef{}, fmt.Errorf("%w: %s/%s/%d", ErrInvalidSeatRef, section, row, number)
	}
	return SeatRef{section: section, row: row, number: number}, nil
}

// ParseSeatRef parses "SECTION/ROW/NUMBER".
func ParseSeatRef(raw string) (SeatRef, error) {
	parts := strings.Split(strings.TrimSpace(raw), "/")
	if len(parts) != 3 {
		return SeatRef{}, fmt.Errorf("%w: %q", ErrInvalidSeatRef, raw)
	}
	n, err := strconv.Atoi(parts[2])
	if err != nil {
		return SeatRef{}, fmt.Errorf("%w: %q", ErrInvalidSeatRef, raw)
	}
	return NewSeatRef(parts[0], parts[1], n)
}

// Section returns the section code.
func (r SeatRef) Section() string { return r.section }

// IsGeneralAdmission reports whether this is a standing place.
func (r SeatRef) IsGeneralAdmission() bool { return r.row == gaRow }

// String returns the canonical form; GA numbers have four digits.
func (r SeatRef) String() string {
	if r.IsGeneralAdmission() {
		return fmt.Sprintf("%s/%s/%04d", r.section, gaRow, r.number)
	}
	return fmt.Sprintf("%s/%s/%d", r.section, r.row, r.number)
}
