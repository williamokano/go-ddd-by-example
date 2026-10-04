package consumer_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driving/consumer"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/contracts"
)

type soldOutStub struct{ got *application.MarkShowSoldOut }

func (s *soldOutStub) Handle(_ context.Context, cmd application.MarkShowSoldOut) error {
	s.got = &cmd
	return nil
}

func TestTicketingConsumer_SoldOutMarksTheShow(t *testing.T) {
	stub := &soldOutStub{}
	c := consumer.NewTicketingConsumer(stub, slog.New(slog.NewTextHandler(io.Discard, nil)))
	payload, _ := json.Marshal(contracts.InventorySoldOutV1{ShowID: "s1"})

	if err := c.Handle(context.Background(), kafka.Envelope{EventType: contracts.TypeInventorySoldOutV1, Payload: payload}); err != nil {
		t.Fatal(err)
	}

	if stub.got == nil || stub.got.ShowID != "s1" {
		t.Errorf("command = %+v", stub.got)
	}
}

func TestTicketingConsumer_IgnoresTheOtherTicketingFacts(t *testing.T) {
	stub := &soldOutStub{}
	c := consumer.NewTicketingConsumer(stub, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := c.Handle(context.Background(), kafka.Envelope{EventType: contracts.TypeTicketsIssuedV1}); err != nil || stub.got != nil {
		t.Errorf("err = %v, called %v", err, stub.got != nil)
	}
}
