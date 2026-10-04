package postgres_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/show/contracts"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

var (
	at      = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
	showID  = domain.NewShowID(uuid.MustParse("0192f5e0-0000-7000-8000-000000000001"))
	venueID = domain.NewVenueID(uuid.MustParse("0192f5e0-0000-7000-8000-000000000002"))
)

func newID() uuid.UUID { return uuid.MustParse("00000000-0000-7000-8000-0000000000e1") }

func TestToOutboxMessages_ShowPublished(t *testing.T) {
	start := at.Add(30 * 24 * time.Hour)
	schedule, _ := domain.NewSchedule(start.Add(-time.Hour), start, start.Add(2*time.Hour))
	eur, _ := sharedkernel.NewCurrency("EUR")
	orch, _ := sharedkernel.NewMoney(4500, eur)
	floor, _ := sharedkernel.NewMoney(2500, eur)
	prices, _ := domain.NewPriceList(map[string]sharedkernel.Money{"ORCH": orch, "FLOOR": floor})
	layout := domain.VenueLayout{VenueID: venueID, Active: true, Sections: []domain.LayoutSection{
		{Code: "ORCH", Kind: "seated", Rows: []domain.LayoutRow{{Label: "A", Seats: 2}}},
		{Code: "FLOOR", Kind: "ga", Capacity: 3},
	}}

	msgs, err := postgres.ToOutboxMessages([]sharedkernel.DomainEvent{
		domain.ShowDrafted{ShowID: showID, At: at}, // internal: not published
		domain.ShowPublished{ShowID: showID, VenueID: venueID, Title: "Fado", Schedule: schedule, Layout: layout, Prices: prices, At: at},
	}, newID)

	if err != nil || len(msgs) != 1 {
		t.Fatalf("got %d messages, %v; want 1", len(msgs), err)
	}
	if msgs[0].Topic != contracts.Topic || msgs[0].Key != showID.String() || msgs[0].Type != contracts.TypeShowPublishedV1 {
		t.Errorf("message = %+v", msgs[0])
	}
	var got contracts.ShowPublishedV1
	if err := json.Unmarshal(msgs[0].Payload, &got); err != nil {
		t.Fatal(err)
	}
	want := contracts.ShowPublishedV1{
		ShowID: showID.String(), VenueID: venueID.String(), Title: "Fado",
		DoorsOpen: start.Add(-time.Hour), StartsAt: start, EndsAt: start.Add(2 * time.Hour), PublishedAt: at,
		Sections: []contracts.SectionV1{
			{Code: "ORCH", Kind: "seated", Rows: []contracts.RowV1{{Label: "A", Seats: 2}}, Price: contracts.PriceV1{Amount: 4500, Currency: "EUR"}},
			{Code: "FLOOR", Kind: "ga", Capacity: 3, Price: contracts.PriceV1{Amount: 2500, Currency: "EUR"}},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("payload mismatch (-want +got):\n%s", diff)
	}
}

func TestToOutboxMessages_ShowCancelled(t *testing.T) {
	reason, _ := domain.NewCancellationReason("venue_retired")

	msgs, err := postgres.ToOutboxMessages([]sharedkernel.DomainEvent{
		domain.ShowCancelled{ShowID: showID, VenueID: venueID, Reason: reason, At: at},
	}, newID)

	if err != nil || len(msgs) != 1 || msgs[0].Type != contracts.TypeShowCancelledV1 {
		t.Fatalf("got %+v, %v", msgs, err)
	}
	var got contracts.ShowCancelledV1
	_ = json.Unmarshal(msgs[0].Payload, &got)
	if got != (contracts.ShowCancelledV1{ShowID: showID.String(), VenueID: venueID.String(), Reason: "venue_retired", CancelledAt: at}) {
		t.Errorf("payload = %+v", got)
	}
}
