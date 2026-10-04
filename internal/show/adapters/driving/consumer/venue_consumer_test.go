package consumer_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driving/consumer"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/contracts"
)

type stubs struct {
	activated *application.OnVenueActivated
	retired   *application.OnVenueRetired
}

type activatedStub struct{ s *stubs }

func (a activatedStub) Handle(_ context.Context, cmd application.OnVenueActivated) error {
	a.s.activated = &cmd
	return nil
}

type retiredStub struct{ s *stubs }

func (r retiredStub) Handle(_ context.Context, cmd application.OnVenueRetired) error {
	r.s.retired = &cmd
	return nil
}

func envelope(t *testing.T, eventType string, payload any) kafka.Envelope {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return kafka.Envelope{EventID: "e1", EventType: eventType, Payload: b}
}

func TestVenueConsumer(t *testing.T) {
	var logs bytes.Buffer
	newConsumer := func(s *stubs) *consumer.VenueConsumer {
		return consumer.NewVenueConsumer(activatedStub{s}, retiredStub{s}, slog.New(slog.NewTextHandler(&logs, nil)))
	}

	t.Run("venue.activated.v1 becomes OnVenueActivated", func(t *testing.T) {
		s := &stubs{}
		env := envelope(t, contracts.TypeVenueActivatedV1, contracts.VenueActivatedV1{
			VenueID: "v1", Name: "Coliseu", Country: "PT", ActivatedAt: time.Now(),
			Sections: []contracts.SectionV1{
				{Code: "ORCH", Kind: contracts.KindSeated, Rows: []contracts.RowV1{{Label: "A", Seats: 10, AccessibleSeats: []int{1}}}},
				{Code: "FLOOR", Kind: contracts.KindGA, Capacity: 500},
			},
		})

		if err := newConsumer(s).Handle(context.Background(), env); err != nil {
			t.Fatal(err)
		}

		want := &application.OnVenueActivated{VenueID: "v1", Name: "Coliseu", Country: "PT", Sections: []application.LayoutSectionSpec{
			{Code: "ORCH", Kind: "seated", Rows: []application.RowSpec{{Label: "A", Seats: 10, Accessible: []int{1}}}},
			{Code: "FLOOR", Kind: "ga", Capacity: 500},
		}}
		if diff := cmp.Diff(want, s.activated); diff != "" {
			t.Errorf("command mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("venue.retired.v1 becomes OnVenueRetired", func(t *testing.T) {
		s := &stubs{}
		env := envelope(t, contracts.TypeVenueRetiredV1, contracts.VenueRetiredV1{VenueID: "v1", RetiredAt: time.Now()})

		if err := newConsumer(s).Handle(context.Background(), env); err != nil {
			t.Fatal(err)
		}

		if s.retired == nil || s.retired.VenueID != "v1" {
			t.Errorf("command = %+v", s.retired)
		}
	})

	t.Run("an unknown event type is logged and skipped, not an error", func(t *testing.T) {
		s := &stubs{}

		err := newConsumer(s).Handle(context.Background(), envelope(t, "venue.renamed.v1", map[string]string{}))

		if err != nil || s.activated != nil || s.retired != nil {
			t.Errorf("err = %v, calls %v %v; want skipped", err, s.activated, s.retired)
		}
		if !strings.Contains(logs.String(), "venue.renamed.v1") {
			t.Errorf("not logged: %q", logs.String())
		}
	})

	t.Run("an undecodable payload is an error (it ends in the DLQ)", func(t *testing.T) {
		env := kafka.Envelope{EventType: contracts.TypeVenueRetiredV1, Payload: json.RawMessage(`"nope"`)}

		if err := newConsumer(&stubs{}).Handle(context.Background(), env); err == nil {
			t.Error("error = nil, want a decode error")
		}
	})
}

// A payload that does not decode is poison: retrying cannot fix it (8.5).
func TestVenueConsumer_AMalformedPayloadIsPermanent(t *testing.T) {
	s := &stubs{}
	c := consumer.NewVenueConsumer(activatedStub{s}, retiredStub{s}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, typ := range []string{contracts.TypeVenueActivatedV1, contracts.TypeVenueRetiredV1} {
		if err := c.Handle(context.Background(), kafka.Envelope{EventType: typ, Payload: []byte("{")}); !kafka.IsPermanent(err) {
			t.Errorf("%s: err = %v, want a permanent error", typ, err)
		}
	}
}
