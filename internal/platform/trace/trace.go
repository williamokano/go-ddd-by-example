// Package trace carries the correlation ID (one per business flow, e.g. one
// purchase) and the causation ID (the event that caused this work) through
// context.Context, and adds both to every log line (8.3).
package trace

import (
	"context"
	"log/slog"
)

type (
	correlationKey struct{}
	causationKey   struct{}
)

// WithCorrelationID returns ctx carrying the flow's correlation ID.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationKey{}, id)
}

// CorrelationID returns the correlation ID in ctx, or "".
func CorrelationID(ctx context.Context) string {
	id, _ := ctx.Value(correlationKey{}).(string)
	return id
}

// WithCausationID returns ctx carrying the ID of the event being handled.
func WithCausationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, causationKey{}, id)
}

// CausationID returns the causation ID in ctx, or "".
func CausationID(ctx context.Context) string {
	id, _ := ctx.Value(causationKey{}).(string)
	return id
}

// Handler wraps a slog.Handler and adds the IDs found in the record's
// context.
type Handler struct{ next slog.Handler }

// NewHandler wraps next.
func NewHandler(next slog.Handler) *Handler { return &Handler{next: next} }

// Enabled implements slog.Handler.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

// Handle implements slog.Handler.
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if id := CorrelationID(ctx); id != "" {
		r.AddAttrs(slog.String("correlation_id", id))
	}
	if id := CausationID(ctx); id != "" {
		r.AddAttrs(slog.String("causation_id", id))
	}
	return h.next.Handle(ctx, r) //nolint:wrapcheck // a decorator: the wrapped handler's error is ours
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{next: h.next.WithAttrs(attrs)}
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{next: h.next.WithGroup(name)}
}
