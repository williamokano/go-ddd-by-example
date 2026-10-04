package domain_test

import (
	"errors"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func TestIDs(t *testing.T) {
	type id interface{ String() string }
	parsers := map[string]func(string) (id, error){
		"ShowID":     func(s string) (id, error) { return domain.ParseShowID(s) },
		"CustomerID": func(s string) (id, error) { return domain.ParseCustomerID(s) },
		"HoldID":     func(s string) (id, error) { return domain.ParseHoldID(s) },
		"OrderID":    func(s string) (id, error) { return domain.ParseOrderID(s) },
	}
	for name, parse := range parsers {
		if got, err := parse("0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f"); err != nil || got.String() != "0192f5e0-7c1a-7b3e-9d2a-3f4b5c6d7e8f" {
			t.Errorf("%s round-trip = %v, %v", name, got, err)
		}
		if _, err := parse("nope"); !errors.Is(err, domain.ErrInvalidID) {
			t.Errorf("%s: error = %v, want %v", name, err, domain.ErrInvalidID)
		}
	}
}
