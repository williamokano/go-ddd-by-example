package domain_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestRegisterVenue_RecordsVenueRegistered(t *testing.T) {
	venue, err := domain.RegisterVenue(aVenueID(), "Coliseu dos Recreios", mustAddress(t), fixedNow)
	if err != nil {
		t.Fatal(err)
	}

	got := venue.PullEvents()

	want := []sharedkernel.DomainEvent{
		domain.VenueRegistered{VenueID: aVenueID(), Name: "Coliseu dos Recreios", At: fixedNow},
	}
	if diff := cmp.Diff(want, got, domainValues); diff != "" {
		t.Errorf("events mismatch (-want +got):\n%s", diff)
	}
}

func TestVenue_AddSection_RecordsSectionAdded(t *testing.T) {
	venue := newDraftVenue(t)
	venue.PullEvents() // discard the creation event
	section := seatedSection(t, "ORCH", mustRow(t, "A", 20))

	if err := venue.AddSection(section, fixedNow); err != nil {
		t.Fatal(err)
	}

	want := []sharedkernel.DomainEvent{
		domain.SectionAdded{VenueID: venue.ID(), Section: section, At: fixedNow},
	}
	if diff := cmp.Diff(want, venue.PullEvents(), domainValues); diff != "" {
		t.Errorf("events mismatch (-want +got):\n%s", diff)
	}
}

func TestVenue_Activate_RecordsVenueActivatedWithTheLayout(t *testing.T) {
	orch := seatedSection(t, "ORCH", mustRow(t, "A", 10), mustRow(t, "B", 12))
	floor := gaSection(t, "FLOOR", 500)
	venue := newDraftVenue(t, withSection(orch), withSection(floor))
	venue.PullEvents()

	if err := venue.Activate(fixedNow); err != nil {
		t.Fatal(err)
	}

	want := []sharedkernel.DomainEvent{domain.VenueActivated{
		VenueID:  venue.ID(),
		Name:     venue.Name(),
		Sections: []domain.Section{orch, floor},
		At:       fixedNow,
	}}
	if diff := cmp.Diff(want, venue.PullEvents(), domainValues); diff != "" {
		t.Errorf("events mismatch (-want +got):\n%s", diff)
	}
}

func TestVenue_Retire_RecordsVenueRetired(t *testing.T) {
	venue := newActiveVenue(t)
	venue.PullEvents()

	if err := venue.Retire(fixedNow); err != nil {
		t.Fatal(err)
	}

	want := []sharedkernel.DomainEvent{domain.VenueRetired{VenueID: venue.ID(), At: fixedNow}}
	if diff := cmp.Diff(want, venue.PullEvents(), domainValues); diff != "" {
		t.Errorf("events mismatch (-want +got):\n%s", diff)
	}
}

func TestVenue_FailedCommandsRecordNothing(t *testing.T) {
	tests := []struct {
		name    string
		venue   func(t *testing.T) *domain.Venue
		command func(t *testing.T, v *domain.Venue) error
	}{
		{"duplicate section (VEN-2)", newDraftVenueWithSection, func(t *testing.T, v *domain.Venue) error {
			return v.AddSection(gaSection(t, "FLOOR", 10), fixedNow)
		}},
		{"section on an active venue (VEN-4)", newActiveVenue, func(t *testing.T, v *domain.Venue) error {
			return v.AddSection(gaSection(t, "BALCONY", 10), fixedNow)
		}},
		{"activate without sections (VEN-5)", func(t *testing.T) *domain.Venue { return newDraftVenue(t) },
			func(_ *testing.T, v *domain.Venue) error { return v.Activate(fixedNow) }},
		{"retire a draft (VEN-6)", newDraftVenueWithSection,
			func(_ *testing.T, v *domain.Venue) error { return v.Retire(fixedNow) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			venue := tt.venue(t)
			venue.PullEvents()

			if err := tt.command(t, venue); err == nil {
				t.Fatal("command succeeded, want an error")
			}

			if got := venue.PullEvents(); len(got) != 0 {
				t.Errorf("recorded %v, want no events", got)
			}
		})
	}
}

func TestVenue_PullEvents(t *testing.T) {
	venue := newDraftVenue(t, withSection(gaSection(t, "FLOOR", 500)))

	first := venue.PullEvents()
	second := venue.PullEvents()

	wantNames := []string{"venue.VenueRegistered", "venue.SectionAdded"}
	if diff := cmp.Diff(wantNames, eventNames(first)); diff != "" {
		t.Errorf("first pull mismatch (-want +got):\n%s", diff)
	}
	if len(second) != 0 {
		t.Errorf("second pull = %v, want nothing", second)
	}
}

func TestDomainEvents(t *testing.T) {
	tests := []struct {
		event    sharedkernel.DomainEvent
		wantName string
	}{
		{domain.VenueRegistered{At: fixedNow}, "venue.VenueRegistered"},
		{domain.SectionAdded{At: fixedNow}, "venue.SectionAdded"},
		{domain.VenueActivated{At: fixedNow}, "venue.VenueActivated"},
		{domain.VenueRetired{At: fixedNow}, "venue.VenueRetired"},
	}
	for _, tt := range tests {
		if got := tt.event.EventName(); got != tt.wantName {
			t.Errorf("EventName() = %q, want %q", got, tt.wantName)
		}
		if got := tt.event.OccurredAt(); !got.Equal(fixedNow) {
			t.Errorf("%s.OccurredAt() = %v, want %v", tt.wantName, got, fixedNow)
		}
	}
}

func eventNames(events []sharedkernel.DomainEvent) []string {
	names := make([]string, len(events))
	for i, ev := range events {
		names[i] = ev.EventName()
	}
	return names
}
