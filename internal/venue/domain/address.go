package domain

import "strings"

// Address is where a venue is (VEN-1).
type Address struct {
	street  string
	city    string
	country string
}

// NewAddress builds an Address.
func NewAddress(street, city, country string) (Address, error) {
	return Address{
		street:  strings.TrimSpace(street),
		city:    strings.TrimSpace(city),
		country: country,
	}, nil
}

// Street returns the street line.
func (a Address) Street() string { return a.street }

// City returns the city.
func (a Address) City() string { return a.city }

// Country returns the ISO-3166 alpha-2 country code, upper case.
func (a Address) Country() string { return a.country }
