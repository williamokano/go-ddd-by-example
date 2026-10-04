package httpapi_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driving/httpapi"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

type checkInStub struct {
	got *application.CheckIn
	err error
}

func (c *checkInStub) Handle(_ context.Context, cmd application.CheckIn) error {
	c.got = &cmd
	return c.err
}

func checkInHandler(c *checkInStub) http.Handler {
	return httpapi.Routes(httpapi.UseCases{CheckIn: c}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestCheckIn(t *testing.T) {
	c := &checkInStub{}

	w := do(checkInHandler(c), http.MethodPost, "/tickets/ABCD-EFGH-IJKL/check-in", `{"gateId":"north-1"}`)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status %d (%s)", w.Code, w.Body)
	}
	if *c.got != (application.CheckIn{TicketCode: "ABCD-EFGH-IJKL", GateID: "north-1"}) {
		t.Errorf("command = %+v", c.got)
	}
}

func TestCheckIn_Errors(t *testing.T) {
	for err, status := range map[error]int{
		application.ErrTicketNotFound: 404, domain.ErrAlreadyCheckedIn: 409, domain.ErrTicketVoided: 409,
		domain.ErrNotShowDay: 409, domain.ErrInvalidID: 422,
	} {
		w := do(checkInHandler(&checkInStub{err: errors.Join(errors.New("check in"), err)}), http.MethodPost,
			"/tickets/ABCD-EFGH-IJKL/check-in", `{"gateId":"north-1"}`)
		if w.Code != status {
			t.Errorf("%v → %d, want %d", err, w.Code, status)
		}
	}
}
