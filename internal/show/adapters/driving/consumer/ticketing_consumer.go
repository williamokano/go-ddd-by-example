package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/contracts"
)

type markSoldOut interface {
	Handle(context.Context, application.MarkShowSoldOut) error
}

type markBackOnSale interface {
	Handle(context.Context, application.MarkShowBackOnSale) error
}

// TicketingConsumer consumes ticketing.events (group "show"). Show conforms to
// two small facts: the inventory sold out (SHW-8), and it has seats again
// (SHW-10).
type TicketingConsumer struct {
	soldOut markSoldOut
	back    markBackOnSale
	logger  *slog.Logger
}

// NewTicketingConsumer wires the consumer.
func NewTicketingConsumer(soldOut markSoldOut, back markBackOnSale, logger *slog.Logger) *TicketingConsumer {
	return &TicketingConsumer{soldOut: soldOut, back: back, logger: logger}
}

// Handle is a kafka.Handler. Every other Ticketing fact is none of Show's
// business.
func (c *TicketingConsumer) Handle(ctx context.Context, env kafka.Envelope) error {
	var err error
	switch env.EventType {
	case contracts.TypeInventorySoldOutV1:
		var e contracts.InventorySoldOutV1
		if err = decode(env, &e); err == nil {
			err = c.soldOut.Handle(ctx, application.MarkShowSoldOut{ShowID: e.ShowID})
		}
	case contracts.TypeInventoryAvailableAgainV1:
		var e contracts.InventoryAvailableAgainV1
		if err = decode(env, &e); err == nil {
			err = c.back.Handle(ctx, application.MarkShowBackOnSale{ShowID: e.ShowID})
		}
	default:
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", env.EventType, err)
	}
	return nil
}

func decode(env kafka.Envelope, v any) error {
	if err := json.Unmarshal(env.Payload, v); err != nil {
		return kafka.Permanent(fmt.Errorf("decode: %w", err))
	}
	return nil
}
