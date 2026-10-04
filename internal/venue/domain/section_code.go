package domain

import "strings"

// SectionCode identifies a section within a venue, e.g. "ORCH" or "BAL-L".
type SectionCode struct{ value string }

// NewSectionCode parses a section code.
func NewSectionCode(raw string) (SectionCode, error) {
	return SectionCode{value: strings.ToUpper(strings.TrimSpace(raw))}, nil
}

// String returns the code, upper case.
func (c SectionCode) String() string { return c.value }
