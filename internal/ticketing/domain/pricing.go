package domain

import (
	"errors"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
)

// ErrNoVATRate means the pricing policy has no VAT rate for the venue's
// country: Stagehand doesn't sell there (yet).
var ErrNoVATRate = errors.New("ticketing: no VAT rate for the venue's country")

// PricingContext is what the fee rules look at: the order's lines and where
// the show takes place.
type PricingContext struct {
	Lines   []OrderLine
	Country string // the venue's ISO country code
}

// FeeSpecification is one predicate over an order (the specification
// pattern): rules are combined from small ones instead of growing an
// if-else chain.
type FeeSpecification interface {
	IsSatisfiedBy(PricingContext) bool
}

type specFunc func(PricingContext) bool

func (f specFunc) IsSatisfiedBy(c PricingContext) bool { return f(c) }

// AnyOrder is satisfied by every order: the fallback rule.
func AnyOrder() FeeSpecification { return specFunc(func(PricingContext) bool { return true }) }

// SeatsAtLeast is satisfied by orders of n seats or more.
func SeatsAtLeast(n int) FeeSpecification {
	return specFunc(func(c PricingContext) bool { return len(c.Lines) >= n })
}

// InSection is satisfied by orders in that section (a hold is one section).
func InSection(code string) FeeSpecification {
	return specFunc(func(c PricingContext) bool { return len(c.Lines) > 0 && c.Lines[0].Seat.Section() == code })
}

// And is satisfied when every spec is.
func And(specs ...FeeSpecification) FeeSpecification {
	return specFunc(func(c PricingContext) bool {
		for _, s := range specs {
			if !s.IsSatisfiedBy(c) {
				return false
			}
		}
		return true
	})
}

// Or is satisfied when any spec is.
func Or(specs ...FeeSpecification) FeeSpecification {
	return specFunc(func(c PricingContext) bool {
		for _, s := range specs {
			if s.IsSatisfiedBy(c) {
				return true
			}
		}
		return false
	})
}

// Not is satisfied when spec is not.
func Not(spec FeeSpecification) FeeSpecification {
	return specFunc(func(c PricingContext) bool { return !spec.IsSatisfiedBy(c) })
}

// FeeRule charges Percent (basis points) of the subtotal, at least Minimum,
// to the orders its specification picks.
type FeeRule struct {
	Name    string
	When    FeeSpecification
	Percent int64
	Minimum sharedkernel.Money
}

// PriceBreakdown is what an order costs, and why (TKT-15).
type PriceBreakdown struct {
	Subtotal sharedkernel.Money // the seats' prices
	Fee      sharedkernel.Money // Stagehand's service fee
	VAT      sharedkernel.Money // on subtotal and fee
	Total    sharedkernel.Money
}

// PricingPolicy is a domain service (9.8): the total of an order depends on
// its lines, on Stagehand's fee rules and on the venue's country. That
// belongs to no aggregate. The Order shouldn't know tax tables, and the
// inventory shouldn't know fees. So the policy computes the breakdown and the
// Order records it.
type PricingPolicy struct {
	Fees []FeeRule        // the first rule whose specification matches wins
	VAT  map[string]int64 // basis points by ISO country code
}

// Price computes the breakdown of lines for a show in country (TKT-15).
func (p PricingPolicy) Price(lines []OrderLine, country string) (PriceBreakdown, error) {
	if len(lines) == 0 {
		return PriceBreakdown{}, fmt.Errorf("%w: no lines to price", ErrInvalidHoldSize)
	}
	rate, ok := p.VAT[country]
	if !ok {
		return PriceBreakdown{}, fmt.Errorf("%w: %q", ErrNoVATRate, country)
	}
	subtotal := lines[0].Price
	for _, l := range lines[1:] {
		var err error
		if subtotal, err = subtotal.Add(l.Price); err != nil {
			return PriceBreakdown{}, fmt.Errorf("subtotal: %w", err)
		}
	}
	fee, err := sharedkernel.NewMoney(0, subtotal.Currency())
	if err != nil {
		return PriceBreakdown{}, fmt.Errorf("fee: %w", err)
	}
	ctx := PricingContext{Lines: lines, Country: country}
	for _, rule := range p.Fees {
		if rule.When.IsSatisfiedBy(ctx) {
			fee = subtotal.Percent(rule.Percent).Max(rule.Minimum)
			break
		}
	}
	taxable, err := subtotal.Add(fee)
	if err != nil {
		return PriceBreakdown{}, fmt.Errorf("fee: %w", err)
	}
	vat := taxable.Percent(rate)
	total, err := taxable.Add(vat)
	if err != nil {
		return PriceBreakdown{}, fmt.Errorf("total: %w", err)
	}
	return PriceBreakdown{Subtotal: subtotal, Fee: fee, VAT: vat, Total: total}, nil
}

// StandardPricing is Stagehand's pricing (TKT-15): a 10% service fee, 7%
// for groups of six or more, never below EUR 1.50; VAT at the venue
// country's rate for live performances. Illustrative rates, not tax advice.
func StandardPricing() PricingPolicy {
	eur, _ := sharedkernel.NewCurrency("EUR")
	minimum, _ := sharedkernel.NewMoney(150, eur)
	return PricingPolicy{
		Fees: []FeeRule{
			{Name: "group", When: SeatsAtLeast(6), Percent: 700, Minimum: minimum},
			{Name: "standard", When: AnyOrder(), Percent: 1000, Minimum: minimum},
		},
		VAT: map[string]int64{"PT": 600, "ES": 2100, "FR": 550, "DE": 700, "NL": 900, "IT": 1000, "BE": 600, "IE": 900},
	}
}
