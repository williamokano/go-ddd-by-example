package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// The whole state machine of Chapter 1 in one table: for every state and
// command, either "ok → new state" or a specific error (SHW-5, SHW-6, SHW-8).
func TestShow_StateMachine(t *testing.T) {
	commands := map[string]func(t *testing.T, s *domain.Show) error{
		"price":      func(t *testing.T, s *domain.Show) error { return s.Price(fullPrices(t), activeLayout(), now) },
		"reschedule": func(t *testing.T, s *domain.Show) error { return s.Reschedule(inAMonth(t), now) },
		"publish":    func(_ *testing.T, s *domain.Show) error { return s.Publish(activeLayout(), now) },
		"cancel":     func(t *testing.T, s *domain.Show) error { return s.Cancel(reason(t, "promoter ill"), now) },
		"sold out":   func(_ *testing.T, s *domain.Show) error { return s.MarkSoldOut(now) },
	}
	type outcome struct {
		state domain.Status
		err   error
	}
	ok := func(s domain.Status) outcome { return outcome{state: s} }
	fails := func(err error) outcome { return outcome{err: err} }

	table := map[domain.Status]map[string]outcome{
		domain.Draft: {
			"price": ok(domain.Draft), "reschedule": ok(domain.Draft), "publish": ok(domain.Published),
			"cancel": ok(domain.Cancelled), "sold out": fails(domain.ErrInvalidShowTransition),
		},
		domain.Published: {
			"price": fails(domain.ErrShowNotDraft), "reschedule": fails(domain.ErrShowNotDraft),
			"publish": fails(domain.ErrInvalidShowTransition), "cancel": ok(domain.Cancelled), "sold out": ok(domain.SoldOut),
		},
		domain.SoldOut: {
			"price": fails(domain.ErrShowNotDraft), "reschedule": fails(domain.ErrShowNotDraft),
			"publish": fails(domain.ErrInvalidShowTransition), "cancel": ok(domain.Cancelled), "sold out": ok(domain.SoldOut),
		},
		domain.Cancelled: {
			"price": fails(domain.ErrShowNotDraft), "reschedule": fails(domain.ErrShowNotDraft),
			"publish": fails(domain.ErrInvalidShowTransition), "cancel": fails(domain.ErrInvalidShowTransition), "sold out": ok(domain.Cancelled),
		},
		domain.Completed: {
			"price": fails(domain.ErrShowNotDraft), "reschedule": fails(domain.ErrShowNotDraft),
			"publish": fails(domain.ErrInvalidShowTransition), "cancel": fails(domain.ErrInvalidShowTransition), "sold out": ok(domain.Completed),
		},
	}
	for state, row := range table {
		for name, want := range row {
			t.Run(state.String()+"/"+name, func(t *testing.T) {
				s := showIn(t, state)

				err := commands[name](t, s)

				if want.err != nil {
					if !errors.Is(err, want.err) {
						t.Errorf("error = %v, want %v", err, want.err)
					}
					if s.Status() != state {
						t.Errorf("status changed to %v on a failed command", s.Status())
					}
					if ev := s.PullEvents(); len(ev) != 0 {
						t.Errorf("failed command recorded %v", ev)
					}
					return
				}
				if err != nil {
					t.Fatalf("error = %v, want ok", err)
				}
				if s.Status() != want.state {
					t.Errorf("status = %v, want %v", s.Status(), want.state)
				}
			})
		}
	}
}

func TestShow_MarkSoldOut_IsIdempotent(t *testing.T) {
	for _, state := range []domain.Status{domain.SoldOut, domain.Cancelled, domain.Completed} {
		t.Run(state.String()+" records nothing (SHW-8)", func(t *testing.T) {
			s := showIn(t, state)

			if err := s.MarkSoldOut(now); err != nil {
				t.Fatal(err)
			}

			if ev := s.PullEvents(); len(ev) != 0 {
				t.Errorf("recorded %v, want nothing: the fact is old news", ev)
			}
		})
	}

	t.Run("published records ShowSoldOut once", func(t *testing.T) {
		s := showIn(t, domain.Published)

		_ = s.MarkSoldOut(now)
		_ = s.MarkSoldOut(now)

		want := []domain.DomainEvent{domain.ShowSoldOut{ShowID: showID, At: now}}
		if diff := cmp.Diff(want, s.PullEvents(), showValues); diff != "" {
			t.Errorf("events mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestShow_Cancel_RecordsTheReason(t *testing.T) {
	s := showIn(t, domain.Published)

	if err := s.Cancel(reason(t, domain.ReasonVenueRetired), now); err != nil {
		t.Fatal(err)
	}

	want := []domain.DomainEvent{domain.ShowCancelled{ShowID: showID, VenueID: venueID, Reason: reason(t, "venue_retired"), At: now}}
	if diff := cmp.Diff(want, s.PullEvents(), showValues); diff != "" {
		t.Errorf("events mismatch (-want +got):\n%s", diff)
	}
	if s.CancellationReason().String() != "venue_retired" {
		t.Errorf("CancellationReason() = %v", s.CancellationReason())
	}
}

func TestShow_Reschedule(t *testing.T) {
	t.Run("records ShowRescheduled (SHW-5)", func(t *testing.T) {
		s := draftShow(t)
		s.PullEvents()
		later, _ := domain.NewSchedule(now.Add(40*24*time.Hour), now.Add(40*24*time.Hour), now.Add(40*24*time.Hour+time.Hour))

		if err := s.Reschedule(later, now); err != nil {
			t.Fatal(err)
		}

		want := []domain.DomainEvent{domain.ShowRescheduled{ShowID: showID, Schedule: later, At: now}}
		if diff := cmp.Diff(want, s.PullEvents(), showValues); diff != "" {
			t.Errorf("events mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("into the past fails (SHW-2)", func(t *testing.T) {
		past, _ := domain.NewSchedule(now.Add(-2*time.Hour), now.Add(-time.Hour), now)

		if err := draftShow(t).Reschedule(past, now); !errors.Is(err, domain.ErrInvalidSchedule) {
			t.Errorf("error = %v, want %v", err, domain.ErrInvalidSchedule)
		}
	})
}

func TestNewCancellationReason(t *testing.T) {
	for _, bad := range []string{"", "   "} {
		if _, err := domain.NewCancellationReason(bad); !errors.Is(err, domain.ErrInvalidCancellationReason) {
			t.Errorf("NewCancellationReason(%q) error = %v", bad, err)
		}
	}
}

func TestRehydrateShow(t *testing.T) {
	s := showIn(t, domain.SoldOut)

	if s.Status() != domain.SoldOut || s.Version() != 3 || s.Prices().IsZero() || len(s.PullEvents()) != 0 {
		t.Errorf("rehydrated show = %v v%d, events %v", s.Status(), s.Version(), s.PullEvents())
	}
}
