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

// TicketingConsumer consumes ticketing.events (group "show"). Show conforms to
// one small fact: the inventory sold out (SHW-8).
type TicketingConsumer struct {
	soldOut markSoldOut
	logger  *slog.Logger
}

// NewTicketingConsumer wires the consumer.
func NewTicketingConsumer(soldOut markSoldOut, logger *slog.Logger) *TicketingConsumer {
	return &TicketingConsumer{soldOut: soldOut, logger: logger}
}

// Handle is a kafka.Handler. Every other Ticketing fact is none of Show's
// business.
func (c *TicketingConsumer) Handle(ctx context.Context, env kafka.Envelope) error {
	if env.EventType != contracts.TypeInventorySoldOutV1 {
		return nil
	}
	var e contracts.InventorySoldOutV1
	if err := json.Unmarshal(env.Payload, &e); err != nil {
		return fmt.Errorf("decode %s: %w", env.EventType, err)
	}
	if err := c.soldOut.Handle(ctx, application.MarkShowSoldOut{ShowID: e.ShowID}); err != nil {
		return fmt.Errorf("%s: %w", env.EventType, err)
	}
	return nil
}
