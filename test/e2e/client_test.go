//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// client is a tiny typed HTTP client for the running app (make up).
type client struct {
	t    *testing.T
	base string
	http *http.Client
}

func newClient(t *testing.T) *client {
	t.Helper()
	base := os.Getenv("STAGEHAND_URL")
	if base == "" {
		base = "http://localhost:8080"
	}
	return &client{t: t, base: base, http: &http.Client{Timeout: 10 * time.Second}}
}

type response struct {
	Status int
	Body   []byte
}

// decode unmarshals the response body into v.
func (r response) decode(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("decode %s: %v", r.Body, err)
	}
}

func (c *client) do(method, path string, body any) response {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(c.t.Context(), method, c.base+path, reader)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v (is the app running? make up)", method, path, err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	return response{Status: resp.StatusCode, Body: b}
}

// mustStatus fails the test unless r has the wanted status.
func mustStatus(t *testing.T, r response, want int) response {
	t.Helper()
	if r.Status != want {
		t.Fatalf("status = %d, want %d: %s", r.Status, want, r.Body)
	}
	return r
}

type venueJSON struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Capacity int    `json:"capacity"`
}

type problemJSON struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
}

func (c *client) registerVenue(name string) string {
	c.t.Helper()
	var created struct{ ID string }
	mustStatus(c.t, c.do(http.MethodPost, "/venues", map[string]string{
		"name": name, "street": "Rua Portas de Santo Antão 96", "city": "Lisboa", "country": "PT",
	}), http.StatusCreated).decode(c.t, &created)
	return created.ID
}
