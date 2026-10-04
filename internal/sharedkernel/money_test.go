package sharedkernel_test

import (
	"errors"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
)

func eurMoney(t *testing.T, minor int64) sharedkernel.Money {
	t.Helper()
	m, err := sharedkernel.NewMoney(minor, mustCurrency(t, "EUR"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func mustCurrency(t *testing.T, code string) sharedkernel.Currency {
	t.Helper()
	c, err := sharedkernel.NewCurrency(code)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewCurrency(t *testing.T) {
	if c, err := sharedkernel.NewCurrency("eur"); err != nil || c.String() != "EUR" {
		t.Errorf("NewCurrency(eur) = %v, %v; want EUR", c, err)
	}
	for _, bad := range []string{"", "EU", "EURO", "E1R"} {
		if _, err := sharedkernel.NewCurrency(bad); !errors.Is(err, sharedkernel.ErrInvalidMoney) {
			t.Errorf("NewCurrency(%q) error = %v, want %v", bad, err, sharedkernel.ErrInvalidMoney)
		}
	}
}

func TestMoney(t *testing.T) {
	t.Run("rejects negative amounts", func(t *testing.T) {
		if _, err := sharedkernel.NewMoney(-1, mustCurrency(t, "EUR")); !errors.Is(err, sharedkernel.ErrInvalidMoney) {
			t.Errorf("error = %v, want %v", err, sharedkernel.ErrInvalidMoney)
		}
	})

	t.Run("adds amounts of the same currency", func(t *testing.T) {
		sum, err := eurMoney(t, 1500).Add(eurMoney(t, 250))

		if err != nil || sum != eurMoney(t, 1750) {
			t.Errorf("Add() = %v, %v; want EUR 17.50", sum, err)
		}
	})

	t.Run("refuses to add different currencies", func(t *testing.T) {
		usd, _ := sharedkernel.NewMoney(100, mustCurrency(t, "USD"))

		if _, err := eurMoney(t, 100).Add(usd); !errors.Is(err, sharedkernel.ErrCurrencyMismatch) {
			t.Errorf("error = %v, want %v", err, sharedkernel.ErrCurrencyMismatch)
		}
	})

	t.Run("prints in major units", func(t *testing.T) {
		for minor, want := range map[int64]string{1500: "EUR 15.00", 5: "EUR 0.05", 0: "EUR 0.00", 123456: "EUR 1234.56"} {
			if got := eurMoney(t, minor).String(); got != want {
				t.Errorf("String(%d) = %q, want %q", minor, got, want)
			}
		}
	})

	t.Run("knows when it is positive", func(t *testing.T) {
		if eurMoney(t, 0).IsPositive() || !eurMoney(t, 1).IsPositive() {
			t.Error("IsPositive() wrong")
		}
	})
}

// Fees and VAT are percentages of money (9.8), in basis points (1% = 100),
// rounded half up to the cent: no float64 anywhere.
func TestMoney_Percent(t *testing.T) {
	for _, tt := range []struct {
		amount, bp, want int64
	}{
		{9000, 1000, 900}, // 10% of 90.00
		{9900, 600, 594},  // 6% of 99.00
		{1250, 550, 69},   // 5.5% of 12.50 = 0.6875 → 0.69
		{1, 5000, 1},      // 50% of 0.01 = 0.005 → 0.01
		{0, 2100, 0},
	} {
		if got := eurMoney(t, tt.amount).Percent(tt.bp); got != eurMoney(t, tt.want) {
			t.Errorf("%d bp of %d = %v, want %d", tt.bp, tt.amount, got, tt.want)
		}
	}
}

func TestMoney_Max(t *testing.T) {
	if got := eurMoney(t, 150).Max(eurMoney(t, 90)); got != eurMoney(t, 150) {
		t.Errorf("Max = %v", got)
	}
	if got := eurMoney(t, 90).Max(eurMoney(t, 150)); got != eurMoney(t, 150) {
		t.Errorf("Max = %v", got)
	}
}
