package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// existingShow rehydrates another show at the venue, in the given status,
// `offset` after the candidate.
func existingShow(t *testing.T, status domain.Status, offset time.Duration) *domain.Show {
	t.Helper()
	base := inAMonth(t)
	schedule, err := domain.NewSchedule(base.DoorsOpen().Add(offset), base.StartsAt().Add(offset), base.EndsAt().Add(offset))
	if err != nil {
		t.Fatal(err)
	}
	return domain.RehydrateShow(domain.ShowState{
		ID: domain.NewShowID(uuid.New()), VenueID: venueID, PromoterID: promoterID,
		Title: "Other", Schedule: schedule, Status: status,
	})
}

func TestSchedulingPolicy_EnsureNoOverlap(t *testing.T) {
	policy := domain.SchedulingPolicy{}

	t.Run("no other shows (SHW-3)", func(t *testing.T) {
		if err := policy.EnsureNoOverlap(showID, inAMonth(t), nil); err != nil {
			t.Errorf("error = %v", err)
		}
	})

	t.Run("an overlapping show is a conflict that names it (SHW-3)", func(t *testing.T) {
		other := existingShow(t, domain.Published, time.Hour)

		err := policy.EnsureNoOverlap(showID, inAMonth(t), []*domain.Show{other})

		if !errors.Is(err, domain.ErrScheduleConflict) || !strings.Contains(err.Error(), other.ID().String()) {
			t.Errorf("error = %v, want %v naming %s", err, domain.ErrScheduleConflict, other.ID())
		}
	})

	t.Run("a cancelled show no longer blocks the slot (SHW-3)", func(t *testing.T) {
		other := existingShow(t, domain.Cancelled, time.Hour)

		if err := policy.EnsureNoOverlap(showID, inAMonth(t), []*domain.Show{other}); err != nil {
			t.Errorf("error = %v", err)
		}
	})

	t.Run("a non-overlapping show is fine", func(t *testing.T) {
		other := existingShow(t, domain.Published, 24*time.Hour)

		if err := policy.EnsureNoOverlap(showID, inAMonth(t), []*domain.Show{other}); err != nil {
			t.Errorf("error = %v", err)
		}
	})

	t.Run("a show being rescheduled does not conflict with itself", func(t *testing.T) {
		self := draftShow(t)

		if err := policy.EnsureNoOverlap(self.ID(), self.Schedule(), []*domain.Show{self}); err != nil {
			t.Errorf("error = %v", err)
		}
	})
}
