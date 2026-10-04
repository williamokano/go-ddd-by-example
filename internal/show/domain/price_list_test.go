package domain_test

import (
	"errors"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

func priceList(t *testing.T, prices map[string]domain.Money) domain.PriceList {
	t.Helper()
	pl, err := domain.NewPriceList(prices)
	if err != nil {
		t.Fatal(err)
	}
	return pl
}

func TestNewPriceList(t *testing.T) {
	t.Run("reads back each section's price, codes upper-cased", func(t *testing.T) {
		pl := priceList(t, map[string]domain.Money{"orch": eur(t, 4500), "FLOOR": eur(t, 2500)})

		if got, ok := pl.Price("ORCH"); !ok || got != eur(t, 4500) {
			t.Errorf("Price(ORCH) = %v, %v", got, ok)
		}
		if pl.Currency().String() != "EUR" {
			t.Errorf("Currency() = %v", pl.Currency())
		}
	})

	tests := map[string]struct {
		prices  map[string]domain.Money
		wantErr error
	}{
		"empty":            {map[string]domain.Money{}, domain.ErrInvalidPriceList},
		"a zero price":     {map[string]domain.Money{"ORCH": eur(t, 0)}, domain.ErrInvalidPriceList},
		"a blank section":  {map[string]domain.Money{" ": eur(t, 100)}, domain.ErrInvalidPriceList},
		"mixed currencies": {map[string]domain.Money{"ORCH": eur(t, 100), "FLOOR": mustMoney(t, 100, "USD")}, domain.ErrCurrencyMismatch},
	}
	for name, tt := range tests {
		t.Run("rejects "+name+" (SHW-4)", func(t *testing.T) {
			if _, err := domain.NewPriceList(tt.prices); !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestPriceList_CoversExactly(t *testing.T) {
	pl := priceList(t, map[string]domain.Money{"ORCH": eur(t, 4500), "FLOOR": eur(t, 2500)})

	tests := map[string]struct {
		sections []string
		ok       bool
	}{
		"exactly the venue's sections": {[]string{"FLOOR", "ORCH"}, true},
		"a section without a price":    {[]string{"FLOOR", "ORCH", "BALCONY"}, false},
		"a price for no section":       {[]string{"ORCH"}, false},
	}
	for name, tt := range tests {
		t.Run(name+" (SHW-4)", func(t *testing.T) {
			err := pl.CoversExactly(tt.sections)

			if tt.ok && err != nil {
				t.Errorf("error = %v, want nil", err)
			}
			if !tt.ok && !errors.Is(err, domain.ErrPriceListMismatch) {
				t.Errorf("error = %v, want %v", err, domain.ErrPriceListMismatch)
			}
		})
	}
}

func mustMoney(t *testing.T, minor int64, currency string) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(minor, mustCurrency(t, currency))
	if err != nil {
		t.Fatal(err)
	}
	return m
}
