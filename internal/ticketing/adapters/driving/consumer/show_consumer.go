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

type (
	openInventory interface {
		Handle(context.Context, application.OpenInventory) error
	}
	closeInventory interface {
		Handle(context.Context, application.CloseInventory) error
	}
)

// ShowConsumer consumes show.events (group "ticketing"). It reads
// show.published.v2 and skips v1 (9.3): both arrive for every show, v1 first.
type ShowConsumer struct {
	open   openInventory
	close  closeInventory
	logger *slog.Logger
}

// NewShowConsumer wires the consumer to Ticketing's use cases.
func NewShowConsumer(open openInventory, closer closeInventory, logger *slog.Logger) *ShowConsumer {
	return &ShowConsumer{open: open, close: closer, logger: logger}
}

// Handle is a kafka.Handler.
func (c *ShowConsumer) Handle(ctx context.Context, env kafka.Envelope) error {
	switch env.EventType {
	case showcontracts.TypeShowPublishedV2:
		var e showcontracts.ShowPublishedV2
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return kafka.Permanent(fmt.Errorf("decode %s: %w", env.EventType, err))
		}
		cmd, err := toOpenInventory(e)
		if err != nil {
			return kafka.Permanent(fmt.Errorf("%s: %w", env.EventType, err))
		}
		if err := c.open.Handle(ctx, cmd); err != nil {
			return fmt.Errorf("%s: %w", env.EventType, err)
		}
		return nil
	case showcontracts.TypeShowCancelledV1:
		var e showcontracts.ShowCancelledV1
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return kafka.Permanent(fmt.Errorf("decode %s: %w", env.EventType, err))
		}
		if err := c.close.Handle(ctx, application.CloseInventory{ShowID: e.ShowID}); err != nil {
			return fmt.Errorf("%s: %w", env.EventType, err)
		}
		return nil
	default:
		c.logger.InfoContext(ctx, "ticketing: skipping show event", "event_type", env.EventType, "event_id", env.EventID)
		return nil
	}
}

// toOpenInventory folds v2's seats back into Ticketing's rows, numbered
// 1..n; a row with gaps is not something Ticketing can sell.
func toOpenInventory(e showcontracts.ShowPublishedV2) (application.OpenInventory, error) {
	cmd := application.OpenInventory{ShowID: e.ShowID, StartsAt: e.StartsAt}
	for _, s := range e.Sections {
		spec := application.SectionSpec{Code: s.Code, Kind: s.Kind, Capacity: s.Capacity, Price: s.Price.Amount, Currency: s.Price.Currency}
		for _, seat := range s.Seats {
			if n := len(spec.Rows); n == 0 || spec.Rows[n-1].Label != seat.Row {
				spec.Rows = append(spec.Rows, application.RowSpec{Label: seat.Row})
			}
			row := &spec.Rows[len(spec.Rows)-1]
			if seat.Number != row.Seats+1 {
				return application.OpenInventory{}, fmt.Errorf("section %s row %s: seat %d after seat %d", s.Code, seat.Row, seat.Number, row.Seats)
			}
			row.Seats++
			if seat.Accessible {
				row.Accessible = append(row.Accessible, seat.Number)
			}
		}
		cmd.Sections = append(cmd.Sections, spec)
	}
	return cmd, nil
}
