package domain

import (
	"fmt"
	"time"
)

// maxShowLength is SHW-2's limit on how long a show runs.
const maxShowLength = 12 * time.Hour

// Schedule is when a show takes place. It occupies the venue from doors open
// until it ends.
type Schedule struct {
	doorsOpen time.Time
	startsAt  time.Time
	endsAt    time.Time
}

// NewSchedule builds a schedule with doorsOpen ≤ startsAt < endsAt, running
// at most 12 hours (SHW-2). Times are stored in UTC.
func NewSchedule(doorsOpen, startsAt, endsAt time.Time) (Schedule, error) {
	switch {
	case doorsOpen.IsZero() || startsAt.IsZero() || endsAt.IsZero():
		return Schedule{}, fmt.Errorf("%w: missing time", ErrInvalidSchedule)
	case doorsOpen.After(startsAt):
		return Schedule{}, fmt.Errorf("%w: doors open after the start", ErrInvalidSchedule)
	case !endsAt.After(startsAt):
		return Schedule{}, fmt.Errorf("%w: ends before it starts", ErrInvalidSchedule)
	case endsAt.Sub(startsAt) > maxShowLength:
		return Schedule{}, fmt.Errorf("%w: longer than %s", ErrInvalidSchedule, maxShowLength)
	}
	return Schedule{doorsOpen: doorsOpen.UTC(), startsAt: startsAt.UTC(), endsAt: endsAt.UTC()}, nil
}

// DoorsOpen is when the audience may enter.
func (s Schedule) DoorsOpen() time.Time { return s.doorsOpen }

// StartsAt is when the show starts.
func (s Schedule) StartsAt() time.Time { return s.startsAt }

// EndsAt is when the show ends.
func (s Schedule) EndsAt() time.Time { return s.endsAt }

// IsZero reports whether the schedule is the zero value.
func (s Schedule) IsZero() bool { return s.startsAt.IsZero() }

// Overlaps reports whether the two schedules occupy the venue at the same
// time, from doors open to end. Touching edges don't overlap (SHW-3).
func (s Schedule) Overlaps(other Schedule) bool {
	return s.doorsOpen.Before(other.endsAt) && other.doorsOpen.Before(s.endsAt)
}
