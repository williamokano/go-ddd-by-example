// Package consumer holds Show's Kafka driving adapters. Each one decodes
// another context's Published Language and translates it into one of Show's
// own commands: the anti-corruption layer. Nothing from the other context's
// contracts goes past this package.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/contracts"
)

type (
	onVenueActivated interface {
		Handle(context.Context, application.OnVenueActivated) error
	}
	onVenueRetired interface {
		Handle(context.Context, application.OnVenueRetired) error
	}
)

// VenueConsumer consumes venue.events (group "show").
type VenueConsumer struct {
	activated onVenueActivated
	retired   onVenueRetired
	logger    *slog.Logger
}

// NewVenueConsumer wires the consumer to Show's policies.
func NewVenueConsumer(activated onVenueActivated, retired onVenueRetired, logger *slog.Logger) *VenueConsumer {
	return &VenueConsumer{activated: activated, retired: retired, logger: logger}
}

// Handle is a kafka.Handler.
func (c *VenueConsumer) Handle(ctx context.Context, env kafka.Envelope) error {
	switch env.EventType {
	case contracts.TypeVenueActivatedV1:
		var e contracts.VenueActivatedV1
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return fmt.Errorf("decode %s: %w", env.EventType, err)
		}
		return c.activated.Handle(ctx, toOnVenueActivated(e))
	case contracts.TypeVenueRetiredV1:
		var e contracts.VenueRetiredV1
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return fmt.Errorf("decode %s: %w", env.EventType, err)
		}
		return c.retired.Handle(ctx, application.OnVenueRetired{VenueID: e.VenueID})
	default:
		c.logger.InfoContext(ctx, "show: skipping unknown venue event", "event_type", env.EventType, "event_id", env.EventID)
		return nil
	}
}

func toOnVenueActivated(e contracts.VenueActivatedV1) application.OnVenueActivated {
	cmd := application.OnVenueActivated{VenueID: e.VenueID, Name: e.Name}
	for _, s := range e.Sections {
		spec := application.LayoutSectionSpec{Code: s.Code, Kind: s.Kind, Capacity: s.Capacity}
		for _, r := range s.Rows {
			spec.Rows = append(spec.Rows, application.RowSpec{Label: r.Label, Seats: r.Seats})
		}
		cmd.Sections = append(cmd.Sections, spec)
	}
	return cmd
}
