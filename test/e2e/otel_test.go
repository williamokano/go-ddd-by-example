//go:build e2e

package e2e_test

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
)

// 9.7: one purchase is one trace in Jaeger, from the HTTP request through
// the outbox and Kafka to the saga's last step.
func TestS1_OnePurchaseIsOneTrace(t *testing.T) {
	jaeger := os.Getenv("STAGEHAND_JAEGER")
	if jaeger == "" {
		jaeger = "http://localhost:16686"
	}
	c := newClient(t)
	show := c.publishedShow()
	customer := uuid.NewString()
	traceID := make([]byte, 16)
	_, _ = rand.Read(traceID)
	traceparent := "00-" + hex.EncodeToString(traceID) + "-00f067aa0ba902b7-01"
	c.checkout(c.hold(show, customer, "ORCH/A/1"), customer, map[string]string{"traceparent": traceparent})

	want := []string{"POST", "consume ticketing.order_paid", "consume ticketing.seats_sold"}
	var names []string
	eventually(t, 40*time.Second, "Jaeger has the purchase's trace", func() bool {
		resp, err := http.Get(jaeger + "/api/traces/" + hex.EncodeToString(traceID))
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()
		var body struct {
			Data []struct {
				Spans []struct {
					OperationName string `json:"operationName"`
				} `json:"spans"`
			} `json:"data"`
		}
		if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&body) != nil || len(body.Data) == 0 {
			return false
		}
		names = names[:0]
		for _, s := range body.Data[0].Spans {
			names = append(names, s.OperationName)
		}
		for _, w := range want {
			if !slices.Contains(names, w) {
				return false
			}
		}
		return true
	})
}
