// Package clock implements the Clock driven port of every context: the real
// clock for production and a fixed, hand-advanced one for tests.
package clock

import (
	"sync"
	"time"
)

// System is the real clock. It reports UTC, so stored and compared times
// never depend on the server's time zone.
type System struct{}

// Now returns the current time in UTC.
func (System) Now() time.Time { return time.Now().UTC() }

// Fixed is a clock that only moves when told to. Tests use it to make
// time-based rules (hold expiry, TKT-4) deterministic: no sleeping.
type Fixed struct {
	mu  sync.Mutex
	now time.Time
}

// NewFixed returns a clock stopped at t.
func NewFixed(t time.Time) *Fixed { return &Fixed{now: t} }

// Now returns the clock's current time.
func (c *Fixed) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward by d.
func (c *Fixed) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
