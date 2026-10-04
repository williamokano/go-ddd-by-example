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

type backStub struct {
	got *application.MarkShowBackOnSale
}

func (s *backStub) Handle(_ context.Context, cmd application.MarkShowBackOnSale) error {
	s.got = &cmd
	return nil
}

// SHW-10 (9.5): seats are on sale again, so a sold-out show is published again.
func TestTicketingConsumer_AvailableAgainPutsTheShowBackOnSale(t *testing.T) {
	back := &backStub{}
	c := consumer.NewTicketingConsumer(&soldOutStub{}, back, slog.New(slog.NewTextHandler(io.Discard, nil)))
	payload, _ := json.Marshal(contracts.InventoryAvailableAgainV1{ShowID: "s1"})

	if err := c.Handle(context.Background(), kafka.Envelope{EventType: contracts.TypeInventoryAvailableAgainV1, Payload: payload}); err != nil {
		t.Fatal(err)
	}

	if back.got == nil || back.got.ShowID != "s1" {
		t.Errorf("command = %+v", back.got)
	}
}

func TestTicketingConsumer_SoldOutMarksTheShow(t *testing.T) {
	stub := &soldOutStub{}
	c := consumer.NewTicketingConsumer(stub, &backStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
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
	c := consumer.NewTicketingConsumer(stub, &backStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if err := c.Handle(context.Background(), kafka.Envelope{EventType: contracts.TypeTicketsIssuedV1}); err != nil || stub.got != nil {
		t.Errorf("err = %v, called %v", err, stub.got != nil)
	}
}

// A payload that does not decode is poison: retrying cannot fix it (8.5).
func TestTicketingConsumer_AMalformedPayloadIsPermanent(t *testing.T) {
	c := consumer.NewTicketingConsumer(&soldOutStub{}, &backStub{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	err := c.Handle(context.Background(), kafka.Envelope{EventType: contracts.TypeInventorySoldOutV1, Payload: []byte("{")})
	if !kafka.IsPermanent(err) {
		t.Errorf("err = %v, want a permanent error", err)
	}
}
