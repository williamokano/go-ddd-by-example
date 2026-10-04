package sharedkernel_test

import (
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
)

type happened struct{ name string }

func (h happened) EventName() string   { return h.name }
func (happened) OccurredAt() time.Time { return time.Time{} }

func TestEvents_PullReturnsInOrderAndForgets(t *testing.T) {
	var events sharedkernel.Events
	events.Record(happened{"first"})
	events.Record(happened{"second"})

	first := events.PullEvents()
	second := events.PullEvents()

	if len(first) != 2 || first[0].EventName() != "first" || first[1].EventName() != "second" || len(second) != 0 {
		t.Errorf("first pull = %v, second pull = %v", first, second)
	}
}
