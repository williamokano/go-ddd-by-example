//go:build e2e

package e2e_test

import (
	"testing"
	"time"
)

// eventually retries check until it returns true or the timeout passes.
// Contexts talk through Kafka, so cross-context effects are eventually
// consistent: tests poll, they never sleep a fixed time.
func eventually(t *testing.T, timeout time.Duration, what string, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !check() {
		if time.Now().After(deadline) {
			t.Fatalf("eventually: %s did not happen within %s", what, timeout)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
