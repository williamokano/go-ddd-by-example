package consumer_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	showcontracts "github.com/williamokano/go-ddd-by-example/internal/show/contracts"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driving/consumer"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
)

type closeStub struct{ got *application.CloseInventory }

func (c *closeStub) Handle(_ context.Context, cmd application.CloseInventory) error {
	c.got = &cmd
	return nil
}

type openStub struct{ got *application.OpenInventory }

func (o *openStub) Handle(_ context.Context, cmd application.OpenInventory) error {
	o.got = &cmd
	return nil
}

// Ticketing reads v2 (9.3): seats come one by one, and the ACL folds them
// back into Ticketing's rows.
func TestShowConsumer_PublishedV2OpensTheInventory(t *testing.T) {
	open := &openStub{}
	c := consumer.NewShowConsumer(open, &closeStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	start := time.Date(2026, 12, 1, 20, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(showcontracts.ShowPublishedV2{
		ShowID: "s1", VenueID: "v1", VenueCountry: "PT", StartsAt: start,
		Sections: []showcontracts.SectionV2{
			{Code: "ORCH", Kind: "seated", Price: showcontracts.PriceV1{Amount: 4500, Currency: "EUR"}, Seats: []showcontracts.SeatV2{
				{Row: "A", Number: 1}, {Row: "A", Number: 2, Accessible: true}, {Row: "B", Number: 1},
			}},
			{Code: "FLOOR", Kind: "ga", Capacity: 3, Price: showcontracts.PriceV1{Amount: 2500, Currency: "EUR"}},
		},
	})

	err := c.Handle(context.Background(), kafka.Envelope{EventType: showcontracts.TypeShowPublishedV2, Payload: payload})

	if err != nil {
		t.Fatal(err)
	}
	want := &application.OpenInventory{ShowID: "s1", Country: "PT", StartsAt: start, Sections: []application.SectionSpec{
		{Code: "ORCH", Kind: "seated", Rows: []application.RowSpec{{Label: "A", Seats: 2, Accessible: []int{2}}, {Label: "B", Seats: 1}},
			Price: 4500, Currency: "EUR"},
		{Code: "FLOOR", Kind: "ga", Capacity: 3, Price: 2500, Currency: "EUR"},
	}}
	if diff := cmp.Diff(want, open.got); diff != "" {
		t.Errorf("command mismatch (-want +got):\n%s", diff)
	}
}

func TestShowConsumer_SkipsUnknownEvents(t *testing.T) {
	open := &openStub{}
	c := consumer.NewShowConsumer(open, &closeStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := c.Handle(context.Background(), kafka.Envelope{EventType: "show.renamed.v1"}); err != nil || open.got != nil {
		t.Errorf("err = %v, called %v", err, open.got != nil)
	}
}

func TestShowConsumer_CancelledClosesTheInventory(t *testing.T) {
	closer := &closeStub{}
	c := consumer.NewShowConsumer(&openStub{}, closer, slog.New(slog.NewTextHandler(io.Discard, nil)))
	payload, _ := json.Marshal(showcontracts.ShowCancelledV1{ShowID: "s1", Reason: "venue_retired"})

	if err := c.Handle(context.Background(), kafka.Envelope{EventType: showcontracts.TypeShowCancelledV1, Payload: payload}); err != nil {
		t.Fatal(err)
	}

	if closer.got == nil || closer.got.ShowID != "s1" {
		t.Errorf("command = %+v", closer.got)
	}
}

// A payload that does not decode is poison: retrying cannot fix it (8.5).
func TestShowConsumer_AMalformedPayloadIsPermanent(t *testing.T) {
	c := consumer.NewShowConsumer(&openStub{}, &closeStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, typ := range []string{showcontracts.TypeShowPublishedV2, showcontracts.TypeShowCancelledV1} {
		if err := c.Handle(context.Background(), kafka.Envelope{EventType: typ, Payload: []byte("{")}); !kafka.IsPermanent(err) {
			t.Errorf("%s: err = %v, want a permanent error", typ, err)
		}
	}
}

// v1 arrives first on the same key and lacks the flags: once Ticketing reads
// v2, it must skip v1, or the inventory would open without them (9.3).
func TestShowConsumer_SkipsPublishedV1(t *testing.T) {
	open := &openStub{}
	c := consumer.NewShowConsumer(open, &closeStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	payload := []byte(`{"show_id":"s1","sections":[]}`) // a v1 record still in the topic

	if err := c.Handle(context.Background(), kafka.Envelope{EventType: showcontracts.TypeShowPublishedV1, Payload: payload}); err != nil || open.got != nil {
		t.Errorf("err = %v, opened = %v; want v1 skipped", err, open.got != nil)
	}
}

func TestShowConsumer_SeatsWithGapsArePermanent(t *testing.T) {
	c := consumer.NewShowConsumer(&openStub{}, &closeStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	payload, _ := json.Marshal(showcontracts.ShowPublishedV2{ShowID: "s1", Sections: []showcontracts.SectionV2{
		{Code: "ORCH", Kind: "seated", Seats: []showcontracts.SeatV2{{Row: "A", Number: 1}, {Row: "A", Number: 3}}},
	}})

	err := c.Handle(context.Background(), kafka.Envelope{EventType: showcontracts.TypeShowPublishedV2, Payload: payload})

	if !kafka.IsPermanent(err) {
		t.Errorf("err = %v, want permanent: Ticketing numbers seats 1..n", err)
	}
}
