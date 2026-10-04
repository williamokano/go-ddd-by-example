package postgres_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/venue/contracts"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

var (
	at      = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
	eventID = uuid.MustParse("00000000-0000-7000-8000-0000000000e1")
	venueID = domain.NewVenueID(uuid.MustParse("0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f"))
)

func fixedID() uuid.UUID { return eventID }

func TestToOutboxMessages_VenueActivated(t *testing.T) {
	code, _ := domain.NewSectionCode("FLOOR")
	floor, _ := domain.NewGeneralAdmissionSection(code, "Floor", 500)

	got, err := postgres.ToOutboxMessages([]sharedkernel.DomainEvent{
		domain.VenueActivated{VenueID: venueID, Name: "Coliseu", Sections: []domain.Section{floor}, At: at},
	}, fixedID)

	if err != nil {
		t.Fatalf("ToOutboxMessages() error = %v", err)
	}
	payload, _ := json.Marshal(contracts.VenueActivatedV1{
		VenueID: venueID.String(), Name: "Coliseu", ActivatedAt: at,
		Sections: []contracts.SectionV1{{Code: "FLOOR", Kind: contracts.KindGA, Capacity: 500}},
	})
	want := []outbox.Message{{
		EventID: eventID, Topic: contracts.Topic, Key: venueID.String(),
		Type: contracts.TypeVenueActivatedV1, Payload: payload, OccurredAt: at,
	}}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("messages mismatch (-want +got):\n%s", diff)
	}
}

// An additive change keeps the version (9.3): consumers that don't know
// accessible_seats ignore it.
func TestToOutboxMessages_VenueActivatedCarriesAccessibleSeats(t *testing.T) {
	code, _ := domain.NewSectionCode("ORCH")
	row, _ := domain.NewRow("A", 10)
	row, _ = row.WithAccessibleSeats(1, 2)
	orch, _ := domain.NewSeatedSection(code, "Orchestra", []domain.Row{row})

	got, err := postgres.ToOutboxMessages([]sharedkernel.DomainEvent{
		domain.VenueActivated{VenueID: venueID, Name: "Coliseu", Sections: []domain.Section{orch}, At: at},
	}, fixedID)

	if err != nil || len(got) != 1 || got[0].Type != contracts.TypeVenueActivatedV1 {
		t.Fatalf("got %+v, %v; want one venue.activated.v1", got, err)
	}
	var payload contracts.VenueActivatedV1
	if err := json.Unmarshal(got[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]int{1, 2}, payload.Sections[0].Rows[0].AccessibleSeats); diff != "" {
		t.Errorf("accessible seats (-want +got):\n%s", diff)
	}
}

func TestToOutboxMessages_VenueRetired(t *testing.T) {
	got, err := postgres.ToOutboxMessages([]sharedkernel.DomainEvent{domain.VenueRetired{VenueID: venueID, At: at}}, fixedID)

	if err != nil || len(got) != 1 || got[0].Type != contracts.TypeVenueRetiredV1 {
		t.Fatalf("got %+v, %v; want one venue.retired.v1", got, err)
	}
	var payload contracts.VenueRetiredV1
	if err := json.Unmarshal(got[0].Payload, &payload); err != nil || payload.VenueID != venueID.String() || !payload.RetiredAt.Equal(at) {
		t.Errorf("payload = %+v, %v", payload, err)
	}
}

// Internal facts nobody outside needs are not published.
func TestToOutboxMessages_InternalEventsStayInside(t *testing.T) {
	got, err := postgres.ToOutboxMessages([]sharedkernel.DomainEvent{
		domain.VenueRegistered{VenueID: venueID, Name: "Coliseu", At: at},
		domain.SectionAdded{VenueID: venueID, At: at},
	}, fixedID)

	if err != nil || len(got) != 0 {
		t.Errorf("got %+v, %v; want no messages", got, err)
	}
}
