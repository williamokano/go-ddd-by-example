package fakegateway

import (
	"context"
	"net/http"
)

// HeaderName is the test-only header that switches the fake's mode for one
// request. It exists because the gateway is a fake: a real provider has no
// such knob, and the composition root would not mount ModeHeader.
const HeaderName = "X-Fake-Payment-Mode"

type modeKey struct{}

// WithMode returns a context in which Charge answers according to mode.
func WithMode(ctx context.Context, mode Mode) context.Context {
	return context.WithValue(ctx, modeKey{}, mode)
}

// ModeHeader is HTTP middleware that reads HeaderName into the request's
// context, so e2e scenarios share one app instance (8.1).
func ModeHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get(HeaderName)
		if raw == "" {
			next.ServeHTTP(w, r)
			return
		}
		mode, err := ParseMode(raw)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithMode(r.Context(), mode)))
	})
}

func (g *Gateway) modeFor(ctx context.Context) Mode {
	if mode, ok := ctx.Value(modeKey{}).(Mode); ok {
		return mode
	}
	return g.mode
}
