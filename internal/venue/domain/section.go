package domain

import (
	"fmt"
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
	code SectionCode
	name string
	kind SectionKind
	rows []Row
}

// NewSeatedSection builds a section of numbered seats, laid out in rows (VEN-3).
func NewSeatedSection(code SectionCode, name string, rows []Row) (Section, error) {
	if len(rows) == 0 {
		return Section{}, fmt.Errorf("%w: seated section %s has no rows", ErrInvalidSection, code)
	}
	return Section{code: code, name: strings.TrimSpace(name), kind: Seated, rows: rows}, nil
}

// Kind returns whether the section is seated or general admission.
func (s Section) Kind() SectionKind { return s.kind }

// Capacity returns the number of places in the section (VEN-7).
func (s Section) Capacity() int {
	total := 0
	for _, r := range s.rows {
		total += r.Seats()
	}
	return total
}
