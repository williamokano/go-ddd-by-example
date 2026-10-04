package domain

import "fmt"

// SchedulingPolicy is a domain service: a rule no single Show can enforce,
// because no show knows the others at its venue (SHW-3). It is pure: the
// application layer loads the venue's shows and passes them in.
type SchedulingPolicy struct{}

// EnsureNoOverlap checks that candidate, the schedule wanted for show id,
// doesn't overlap any other non-cancelled show in existing.
func (SchedulingPolicy) EnsureNoOverlap(id ShowID, candidate Schedule, existing []*Show) error {
	for _, other := range existing {
		if other.ID() == id || other.Status() == Cancelled {
			continue
		}
		if candidate.Overlaps(other.Schedule()) {
			return fmt.Errorf("%w: overlaps show %s (%q)", ErrScheduleConflict, other.ID(), other.Title())
		}
	}
	return nil
}
