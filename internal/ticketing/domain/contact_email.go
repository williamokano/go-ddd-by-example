package domain

import (
	"fmt"
	"regexp"
	"strings"
)

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// ContactEmail is where an order's tickets and any refund notice go (TKT-6).
// The customer types it at checkout; it is not an account.
type ContactEmail struct{ value string }

// NewContactEmail parses an email: trimmed, lower-cased, shaped like
// name@domain.tld. Deliverability is not the domain's business.
func NewContactEmail(raw string) (ContactEmail, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if !emailPattern.MatchString(v) {
		return ContactEmail{}, fmt.Errorf("%w: %q", ErrInvalidContactEmail, raw)
	}
	return ContactEmail{value: v}, nil
}

// String returns the address.
func (e ContactEmail) String() string { return e.value }

// PaymentRef is the payment provider's reference for a charge.
type PaymentRef struct{ value string }

// NewPaymentRef wraps the provider's reference.
func NewPaymentRef(raw string) (PaymentRef, error) {
	if strings.TrimSpace(raw) == "" {
		return PaymentRef{}, fmt.Errorf("%w: blank payment reference", ErrInvalidID)
	}
	return PaymentRef{value: raw}, nil
}

// String returns the reference.
func (r PaymentRef) String() string { return r.value }
