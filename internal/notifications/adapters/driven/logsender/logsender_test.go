package logsender_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/notifications/adapters/driven/logsender"
	"github.com/williamokano/go-ddd-by-example/internal/notifications/application"
)

func TestSender_LogsTheEmail(t *testing.T) {
	var buf bytes.Buffer
	s := logsender.New(slog.New(slog.NewTextHandler(&buf, nil)))

	err := s.Send(context.Background(), application.Email{To: "ana@example.com", Subject: "Your tickets", Body: "..."})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`msg="email sent"`, "to=ana@example.com", `subject="Your tickets"`} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log %q lacks %s", buf.String(), want)
		}
	}
}
