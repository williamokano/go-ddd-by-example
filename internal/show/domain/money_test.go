package domain_test

import (
	"errors"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

func eur(t *testing.T, minor int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(minor, mustCurrency(t, "EUR"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func mustCurrency(t *testing.T, code string) domain.Currency {
	t.Helper()
	c, err := domain.NewCurrency(code)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewCurrency(t *testing.T) {
	if c, err := domain.NewCurrency("eur"); err != nil || c.String() != "EUR" {
		t.Errorf("NewCurrency(eur) = %v, %v; want EUR", c, err)
	}
	for _, bad := range []string{"", "EU", "EURO", "E1R"} {
		if _, err := domain.NewCurrency(bad); !errors.Is(err, domain.ErrInvalidMoney) {
			t.Errorf("NewCurrency(%q) error = %v, want %v", bad, err, domain.ErrInvalidMoney)
		}
	}
}

func TestMoney(t *testing.T) {
	t.Run("rejects negative amounts", func(t *testing.T) {
		if _, err := domain.NewMoney(-1, mustCurrency(t, "EUR")); !errors.Is(err, domain.ErrInvalidMoney) {
			t.Errorf("error = %v, want %v", err, domain.ErrInvalidMoney)
		}
	})

	t.Run("adds amounts of the same currency", func(t *testing.T) {
		sum, err := eur(t, 1500).Add(eur(t, 250))

		if err != nil || sum != eur(t, 1750) {
			t.Errorf("Add() = %v, %v; want EUR 17.50", sum, err)
		}
	})

	t.Run("refuses to add different currencies", func(t *testing.T) {
		usd, _ := domain.NewMoney(100, mustCurrency(t, "USD"))

		if _, err := eur(t, 100).Add(usd); !errors.Is(err, domain.ErrCurrencyMismatch) {
			t.Errorf("error = %v, want %v", err, domain.ErrCurrencyMismatch)
		}
	})

	t.Run("prints in major units", func(t *testing.T) {
		for minor, want := range map[int64]string{1500: "EUR 15.00", 5: "EUR 0.05", 0: "EUR 0.00", 123456: "EUR 1234.56"} {
			if got := eur(t, minor).String(); got != want {
				t.Errorf("String(%d) = %q, want %q", minor, got, want)
			}
		}
	})

	t.Run("knows when it is positive", func(t *testing.T) {
		if eur(t, 0).IsPositive() || !eur(t, 1).IsPositive() {
			t.Error("IsPositive() wrong")
		}
	})
}
