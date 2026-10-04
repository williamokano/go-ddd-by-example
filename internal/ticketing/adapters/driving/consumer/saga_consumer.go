package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/sagamsg"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
)

type (
	confirmHold interface {
		Handle(context.Context, application.ConfirmHold) error
	}
	issueTickets interface {
		Handle(context.Context, application.IssueTickets) error
	}
	refundOrder interface {
		Handle(context.Context, application.RefundOrder) error
	}
)

// SagaSteps are the use cases the checkout saga's messages drive.
type SagaSteps struct {
	Confirm confirmHold
	Issue   issueTickets
	Refund  refundOrder
}

// SagaConsumer consumes ticketing.internal (group "ticketing-saga"):
// choreography, each step reacting to the previous one's fact (ADR-010).
type SagaConsumer struct {
	steps  SagaSteps
	logger *slog.Logger
}

// NewSagaConsumer wires the consumer to the saga's steps.
func NewSagaConsumer(steps SagaSteps, logger *slog.Logger) *SagaConsumer {
	return &SagaConsumer{steps: steps, logger: logger}
}

// Handle is a kafka.Handler.
func (c *SagaConsumer) Handle(ctx context.Context, env kafka.Envelope) error {
	var err error
	switch env.EventType {
	case sagamsg.TypeOrderPaid:
		var m sagamsg.OrderPaid
		if err = json.Unmarshal(env.Payload, &m); err == nil {
			err = c.steps.Confirm.Handle(ctx, application.ConfirmHold{ShowID: m.ShowID, HoldID: m.HoldID, OrderID: m.OrderID})
		}
	case sagamsg.TypeSeatsSold:
		var m sagamsg.SeatsSold
		if err = json.Unmarshal(env.Payload, &m); err == nil {
			err = c.steps.Issue.Handle(ctx, application.IssueTickets{ShowID: m.ShowID, OrderID: m.OrderID, Seats: m.Seats})
		}
	case sagamsg.TypeHoldConfirmationFailed:
		var m sagamsg.HoldConfirmationFailed
		if err = json.Unmarshal(env.Payload, &m); err == nil {
			err = c.steps.Refund.Handle(ctx, application.RefundOrder{OrderID: m.OrderID})
		}
	default:
		c.logger.InfoContext(ctx, "ticketing saga: skipping message", "event_type", env.EventType, "event_id", env.EventID)
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", env.EventType, err)
	}
	return nil
}
