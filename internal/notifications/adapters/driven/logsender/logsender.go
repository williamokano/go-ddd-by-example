// Package logsender is the EmailSender adapter for this project: it logs
// the email instead of delivering it. A real SMTP or provider adapter would
// implement the same port.
package logsender

import (
	"context"
	"log/slog"

	"github.com/williamokano/go-ddd-by-example/internal/notifications/application"
)

// Sender logs emails.
type Sender struct{ logger *slog.Logger }

// New returns a Sender writing to logger.
func New(logger *slog.Logger) *Sender { return &Sender{logger: logger} }

// Send implements application.EmailSender.
func (s *Sender) Send(ctx context.Context, e application.Email) error {
	s.logger.InfoContext(ctx, "email sent", "to", e.To, "subject", e.Subject, "body", e.Body)
	return nil
}
