package consumer_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driving/consumer"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/sagamsg"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
)

type sagaStubs struct {
	confirmed *application.ConfirmHold
	issued    *application.IssueTickets
	refunded  *application.RefundOrder
}

type confirmStub struct{ s *sagaStubs }

func (c confirmStub) Handle(_ context.Context, cmd application.ConfirmHold) error {
	c.s.confirmed = &cmd
	return nil
}

type issueStub struct{ s *sagaStubs }

func (i issueStub) Handle(_ context.Context, cmd application.IssueTickets) error {
	i.s.issued = &cmd
	return nil
}

type refundStub struct{ s *sagaStubs }

func (r refundStub) Handle(_ context.Context, cmd application.RefundOrder) error {
	r.s.refunded = &cmd
	return nil
}

func TestSagaConsumer_RoutesEachStep(t *testing.T) {
	s := &sagaStubs{}
	c := consumer.NewSagaConsumer(consumer.SagaSteps{Confirm: confirmStub{s}, Issue: issueStub{s}, Refund: refundStub{s}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	send := func(eventType string, payload any) {
		b, _ := json.Marshal(payload)
		if err := c.Handle(context.Background(), kafka.Envelope{EventType: eventType, Payload: b}); err != nil {
			t.Fatal(err)
		}
	}

	send(sagamsg.TypeOrderPaid, sagamsg.OrderPaid{OrderID: "o1", ShowID: "s1", HoldID: "h1"})
	send(sagamsg.TypeSeatsSold, sagamsg.SeatsSold{OrderID: "o1", ShowID: "s1", Seats: []string{"ORCH/A/1"}})
	send(sagamsg.TypeHoldConfirmationFailed, sagamsg.HoldConfirmationFailed{OrderID: "o2", ShowID: "s1"})

	if *s.confirmed != (application.ConfirmHold{ShowID: "s1", HoldID: "h1", OrderID: "o1"}) {
		t.Errorf("confirm = %+v", s.confirmed)
	}
	if s.issued == nil || s.issued.OrderID != "o1" || len(s.issued.Seats) != 1 {
		t.Errorf("issue = %+v", s.issued)
	}
	if s.refunded == nil || s.refunded.OrderID != "o2" {
		t.Errorf("refund = %+v", s.refunded)
	}
}

// A payload that does not decode is poison: retrying cannot fix it (8.5).
func TestSagaConsumer_AMalformedPayloadIsPermanent(t *testing.T) {
	s := &sagaStubs{}
	c := consumer.NewSagaConsumer(consumer.SagaSteps{Confirm: confirmStub{s}, Issue: issueStub{s}, Refund: refundStub{s}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, typ := range []string{sagamsg.TypeOrderPaid, sagamsg.TypeSeatsSold, sagamsg.TypeHoldConfirmationFailed, sagamsg.TypeInventoryClosed} {
		if err := c.Handle(context.Background(), kafka.Envelope{EventType: typ, Payload: []byte("{")}); !kafka.IsPermanent(err) {
			t.Errorf("%s: err = %v, want a permanent error", typ, err)
		}
	}
}
