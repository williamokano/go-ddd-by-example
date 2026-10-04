package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// maxBodyBytes caps request bodies: nothing in this API needs more.
const maxBodyBytes = 1 << 20

// DecodeJSON decodes the request body into dst, strictly: unknown fields,
// trailing data and bodies over 1 MiB are rejected. On failure it writes a
// 400 problem and returns false, so handlers can simply return.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	if err == nil && dec.Decode(&struct{}{}) != io.EOF {
		err = errors.New("body must contain a single JSON value")
	}
	if err != nil {
		WriteProblem(w, http.StatusBadRequest, "Malformed request body", err.Error())
		return false
	}
	return true
}
