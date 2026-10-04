package domain

import (
	"fmt"
	"slices"
	"strings"
)

// SectionKind says whether a section has seats or is general admission.
type SectionKind uint8

// The two shapes a section can take.
const (
	Seated SectionKind = iota + 1
	GeneralAdmission
)

// Section is a named area of a venue. It is an entity whose identity, its
// SectionCode, only has to be unique within its venue (VEN-2).
type Section struct {
	code       SectionCode
	name       string
	kind       SectionKind
	rows       []Row
	gaCapacity int
}

// NewSeatedSection builds a section of numbered seats, laid out in rows (VEN-3).
func NewSeatedSection(code SectionCode, name string, rows []Row) (Section, error) {
	if len(rows) == 0 {
		return Section{}, fmt.Errorf("%w: seated section %s has no rows", ErrInvalidSection, code)
	}
	seen := make(map[string]bool, len(rows))
	for _, r := range rows {
		if seen[r.Label()] {
			return Section{}, fmt.Errorf("%w: row %s in section %s", ErrDuplicateRowLabel, r.Label(), code)
		}
		seen[r.Label()] = true
	}
	return Section{code: code, name: strings.TrimSpace(name), kind: Seated, rows: slices.Clone(rows)}, nil
}

// NewGeneralAdmissionSection builds a standing section with a capacity instead
// of seats (VEN-3).
func NewGeneralAdmissionSection(code SectionCode, name string, capacity int) (Section, error) {
	if capacity < 1 {
		return Section{}, fmt.Errorf("%w: general admission section %s has capacity %d", ErrInvalidSection, code, capacity)
	}
	return Section{code: code, name: strings.TrimSpace(name), kind: GeneralAdmission, gaCapacity: capacity}, nil
}

// Code returns the section's identity within its venue.
func (s Section) Code() SectionCode { return s.code }

// Name returns the section's display name.
func (s Section) Name() string { return s.name }

// Kind returns whether the section is seated or general admission.
func (s Section) Kind() SectionKind { return s.kind }

// Rows returns a copy of the rows of a seated section; a general admission
// section has none.
func (s Section) Rows() []Row { return slices.Clone(s.rows) }

// Capacity returns the number of places in the section: its seats, or its
// general admission capacity (VEN-7).
func (s Section) Capacity() int {
	if s.kind == GeneralAdmission {
		return s.gaCapacity
	}
	total := 0
	for _, r := range s.rows {
		total += r.Seats()
	}
	return total
}
