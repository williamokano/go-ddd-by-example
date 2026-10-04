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

func TestShowConsumer_PublishedOpensTheInventory(t *testing.T) {
	open := &openStub{}
	c := consumer.NewShowConsumer(open, &closeStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	start := time.Date(2026, 12, 1, 20, 0, 0, 0, time.UTC)
	payload, _ := json.Marshal(showcontracts.ShowPublishedV1{
		ShowID: "s1", VenueID: "v1", StartsAt: start,
		Sections: []showcontracts.SectionV1{
			{Code: "ORCH", Kind: "seated", Rows: []showcontracts.RowV1{{Label: "A", Seats: 2}}, Price: showcontracts.PriceV1{Amount: 4500, Currency: "EUR"}},
			{Code: "FLOOR", Kind: "ga", Capacity: 3, Price: showcontracts.PriceV1{Amount: 2500, Currency: "EUR"}},
		},
	})

	err := c.Handle(context.Background(), kafka.Envelope{EventType: showcontracts.TypeShowPublishedV1, Payload: payload})

	if err != nil {
		t.Fatal(err)
	}
	want := &application.OpenInventory{ShowID: "s1", StartsAt: start, Sections: []application.SectionSpec{
		{Code: "ORCH", Kind: "seated", Rows: []application.RowSpec{{Label: "A", Seats: 2}}, Price: 4500, Currency: "EUR"},
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
