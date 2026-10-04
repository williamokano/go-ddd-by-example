package clock_test

import (
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/platform/clock"
)

func TestFixed(t *testing.T) {
	at := time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
	c := clock.NewFixed(at)

	if got := c.Now(); !got.Equal(at) {
		t.Errorf("Now() = %v, want %v", got, at)
	}

	c.Advance(10 * time.Minute)

	if got, want := c.Now(), at.Add(10*time.Minute); !got.Equal(want) {
		t.Errorf("Now() after Advance = %v, want %v", got, want)
	}
}

func TestSystem(t *testing.T) {
	before := time.Now()

	got := clock.System{}.Now()

	if got.Before(before) || got.After(time.Now()) {
		t.Errorf("Now() = %v, not the current time", got)
	}
	if got.Location() != time.UTC {
		t.Errorf("Now() location = %v, want UTC", got.Location())
	}
}
