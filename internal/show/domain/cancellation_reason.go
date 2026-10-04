package domain

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ReasonVenueRetired is the reason Show gives when it cancels a show because
// its venue retired (SHW-7).
const ReasonVenueRetired = "venue_retired"

// CancellationReason says why a show was cancelled.
type CancellationReason struct{ value string }

// NewCancellationReason builds a reason of 1–200 characters.
func NewCancellationReason(raw string) (CancellationReason, error) {
	v := strings.TrimSpace(raw)
	if v == "" || utf8.RuneCountInString(v) > 200 {
		return CancellationReason{}, fmt.Errorf("%w: %q", ErrInvalidCancellationReason, raw)
	}
	return CancellationReason{value: v}, nil
}

// String returns the reason.
func (r CancellationReason) String() string { return r.value }
