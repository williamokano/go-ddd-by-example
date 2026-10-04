// Package consumer holds Ticketing's Kafka driving adapters: each translates
// another context's Published Language into a Ticketing command (the ACL).
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	showcontracts "github.com/williamokano/go-ddd-by-example/internal/show/contracts"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
)

type openInventory interface {
	Handle(context.Context, application.OpenInventory) error
}

// ShowConsumer consumes show.events (group "ticketing").
type ShowConsumer struct {
	open   openInventory
	logger *slog.Logger
}

// NewShowConsumer wires the consumer to Ticketing's use cases.
func NewShowConsumer(open openInventory, logger *slog.Logger) *ShowConsumer {
	return &ShowConsumer{open: open, logger: logger}
}

// Handle is a kafka.Handler.
func (c *ShowConsumer) Handle(ctx context.Context, env kafka.Envelope) error {
	switch env.EventType {
	case showcontracts.TypeShowPublishedV1:
		var e showcontracts.ShowPublishedV1
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return fmt.Errorf("decode %s: %w", env.EventType, err)
		}
		if err := c.open.Handle(ctx, toOpenInventory(e)); err != nil {
			return fmt.Errorf("%s: %w", env.EventType, err)
		}
		return nil
	default:
		c.logger.InfoContext(ctx, "ticketing: skipping show event", "event_type", env.EventType, "event_id", env.EventID)
		return nil
	}
}

func toOpenInventory(e showcontracts.ShowPublishedV1) application.OpenInventory {
	cmd := application.OpenInventory{ShowID: e.ShowID, StartsAt: e.StartsAt}
	for _, s := range e.Sections {
		spec := application.SectionSpec{Code: s.Code, Kind: s.Kind, Capacity: s.Capacity, Price: s.Price.Amount, Currency: s.Price.Currency}
		for _, r := range s.Rows {
			spec.Rows = append(spec.Rows, application.RowSpec{Label: r.Label, Seats: r.Seats})
		}
		cmd.Sections = append(cmd.Sections, spec)
	}
	return cmd
}
