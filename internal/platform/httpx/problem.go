// Package httpx is the HTTP plumbing shared by every context's driving
// adapter: strict JSON decoding, JSON and RFC 9457 problem responses, and
// middleware. It is technical, not domain, so it lives in platform.
package httpx

import (
	"encoding/json"
	"net/http"
)

// Problem is an RFC 9457 problem detail (application/problem+json).
type Problem struct {
	Type   string `json:"type,omitempty"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// WriteProblem writes a problem response.
func WriteProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Problem{Title: title, Status: status, Detail: detail})
}

// WriteJSON writes v as a JSON response.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
