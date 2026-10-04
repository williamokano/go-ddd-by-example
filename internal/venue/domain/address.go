package domain

import (
	"fmt"
	"strings"
)

// Address is where a venue is (VEN-1).
type Address struct {
	street  string
	city    string
	country string
}

// NewAddress builds an Address.
func NewAddress(street, city, country string) (Address, error) {
	street = strings.TrimSpace(street)
	city = strings.TrimSpace(city)
	country = strings.TrimSpace(country)

	if street == "" {
		return Address{}, fmt.Errorf("%w: street is blank", ErrInvalidAddress)
	}
	if city == "" {
		return Address{}, fmt.Errorf("%w: city is blank", ErrInvalidAddress)
	}
	if !isTwoLetters(country) {
		return Address{}, fmt.Errorf("%w: country %q is not a two-letter code", ErrInvalidAddress, country)
	}
	return Address{street: street, city: city, country: country}, nil
}

// Street returns the street line.
func (a Address) Street() string { return a.street }

// City returns the city.
func (a Address) City() string { return a.city }

// Country returns the ISO-3166 alpha-2 country code, upper case.
func (a Address) Country() string { return a.country }

func isTwoLetters(s string) bool {
	if len(s) != 2 {
		return false
	}
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}
