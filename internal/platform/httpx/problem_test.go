package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/httpx"
)

func TestWriteProblem(t *testing.T) {
	w := httptest.NewRecorder()

	httpx.WriteProblem(w, http.StatusConflict, "Venue is not a draft", "venue is active")

	p := assertProblem(t, w, http.StatusConflict)
	if p.Title != "Venue is not a draft" || p.Detail != "venue is active" {
		t.Errorf("problem = %+v", p)
	}
}
