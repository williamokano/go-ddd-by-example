package domain

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
)

// PriceList is one price per section code of the venue (SHW-4). Show prices
// sections, not seats.
type PriceList struct{ prices map[string]sharedkernel.Money }

// NewPriceList builds a list of strictly positive prices, all in one currency.
func NewPriceList(prices map[string]sharedkernel.Money) (PriceList, error) {
	if len(prices) == 0 {
		return PriceList{}, fmt.Errorf("%w: no prices", ErrInvalidPriceList)
	}
	out := make(map[string]sharedkernel.Money, len(prices))
	var currency sharedkernel.Currency
	for raw, price := range prices {
		code := strings.ToUpper(strings.TrimSpace(raw))
		if code == "" {
			return PriceList{}, fmt.Errorf("%w: blank section code", ErrInvalidPriceList)
		}
		if !price.IsPositive() {
			return PriceList{}, fmt.Errorf("%w: section %s costs %s", ErrInvalidPriceList, code, price)
		}
		if currency == (sharedkernel.Currency{}) {
			currency = price.Currency()
		}
		if price.Currency() != currency {
			return PriceList{}, fmt.Errorf("%w: %s and %s in one price list", sharedkernel.ErrCurrencyMismatch, currency, price.Currency())
		}
		out[code] = price
	}
	return PriceList{prices: out}, nil
}

// Price returns the price of a section.
func (p PriceList) Price(sectionCode string) (sharedkernel.Money, bool) {
	m, ok := p.prices[strings.ToUpper(sectionCode)]
	return m, ok
}

// Sections returns the priced section codes, sorted.
func (p PriceList) Sections() []string { return slices.Sorted(maps.Keys(p.prices)) }

// Currency returns the list's single currency.
func (p PriceList) Currency() sharedkernel.Currency {
	for _, m := range p.prices {
		return m.Currency()
	}
	return sharedkernel.Currency{}
}

// IsZero reports whether no price list was set.
func (p PriceList) IsZero() bool { return len(p.prices) == 0 }

// CoversExactly checks the list prices every given section and nothing else
// (SHW-4).
func (p PriceList) CoversExactly(sectionCodes []string) error {
	want := make(map[string]bool, len(sectionCodes))
	for _, c := range sectionCodes {
		want[strings.ToUpper(c)] = true
		if _, ok := p.prices[strings.ToUpper(c)]; !ok {
			return fmt.Errorf("%w: section %s has no price", ErrPriceListMismatch, c)
		}
	}
	for code := range p.prices {
		if !want[code] {
			return fmt.Errorf("%w: %s is not a section of the venue", ErrPriceListMismatch, code)
		}
	}
	return nil
}
