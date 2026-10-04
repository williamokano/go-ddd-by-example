package domain

import (
	"fmt"
	"regexp"
	"strings"
)

// sectionCodePattern is VEN-2's format: 1–10 characters of A-Z, 0-9 and '-'.
var sectionCodePattern = regexp.MustCompile(`^[A-Z0-9-]{1,10}$`)

// SectionCode identifies a section within a venue, e.g. "ORCH" or "BAL-L".
// It is unique within its venue, compared case-insensitively (VEN-2), which is
// why it is stored upper case.
type SectionCode struct{ value string }

// NewSectionCode parses a section code: trimmed, upper-cased, then validated.
func NewSectionCode(raw string) (SectionCode, error) {
	v := strings.ToUpper(strings.TrimSpace(raw))
	if !sectionCodePattern.MatchString(v) {
		return SectionCode{}, fmt.Errorf("%w: %q", ErrInvalidSectionCode, raw)
	}
	return SectionCode{value: v}, nil
}

// String returns the code, upper case.
func (c SectionCode) String() string { return c.value }
