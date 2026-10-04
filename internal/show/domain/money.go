package domain

import (
	"fmt"
	"strings"
)

// Currency is an ISO-4217 code, e.g. "EUR".
type Currency struct{ code string }

// NewCurrency parses a three-letter currency code.
func NewCurrency(code string) (Currency, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) != 3 || strings.Trim(code, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "" {
		return Currency{}, fmt.Errorf("%w: currency %q", ErrInvalidMoney, code)
	}
	return Currency{code: code}, nil
}

// String returns the code.
func (c Currency) String() string { return c.code }

// Money is an amount in minor units (cents) of one currency. Never float64:
// 0.1 + 0.2 != 0.3.
type Money struct {
	amount   int64
	currency Currency
}

// NewMoney builds a non-negative amount.
func NewMoney(minor int64, currency Currency) (Money, error) {
	if minor < 0 {
		return Money{}, fmt.Errorf("%w: negative amount %d", ErrInvalidMoney, minor)
	}
	if currency == (Currency{}) {
		return Money{}, fmt.Errorf("%w: no currency", ErrInvalidMoney)
	}
	return Money{amount: minor, currency: currency}, nil
}

// Amount returns the amount in minor units.
func (m Money) Amount() int64 { return m.amount }

// Currency returns the currency.
func (m Money) Currency() Currency { return m.currency }

// IsPositive reports whether the amount is above zero.
func (m Money) IsPositive() bool { return m.amount > 0 }

// Add returns m + other; both must be in the same currency.
func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("%w: %s + %s", ErrCurrencyMismatch, m.currency, other.currency)
	}
	return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}

// String formats the amount in major units, e.g. "EUR 15.00".
func (m Money) String() string {
	return fmt.Sprintf("%s %d.%02d", m.currency, m.amount/100, m.amount%100)
}
