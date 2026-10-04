package domain_test

import (
	"errors"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// lines prices n FLOOR places at EUR 25 and ORCH seats at EUR 45.
func lines(t *testing.T, seats ...string) []domain.OrderLine {
	t.Helper()
	out := make([]domain.OrderLine, len(seats))
	for i, ref := range refs(t, seats...) {
		price := eur(t, 4500)
		if ref.IsGeneralAdmission() {
			price = eur(t, 2500)
		}
		out[i] = domain.OrderLine{Seat: ref, Price: price}
	}
	return out
}

// The fee rules are specifications (9.8): small predicates over an order,
// combined with And/Or/Not, so a new rule is a new value, not a new branch.
func TestFeeSpecifications(t *testing.T) {
	two := domain.PricingContext{Lines: lines(t, "ORCH/A/1", "ORCH/A/2"), Country: "PT"}
	six := domain.PricingContext{Lines: lines(t, "FLOOR/GA/0001", "FLOOR/GA/0002", "FLOOR/GA/0003",
		"FLOOR/GA/0004", "FLOOR/GA/0005", "FLOOR/GA/0006"), Country: "PT"}

	for name, tt := range map[string]struct {
		spec domain.FeeSpecification
		ctx  domain.PricingContext
		want bool
	}{
		"any order":                     {domain.AnyOrder(), two, true},
		"6+ seats, given 2":             {domain.SeatsAtLeast(6), two, false},
		"6+ seats, given 6":             {domain.SeatsAtLeast(6), six, true},
		"in FLOOR, given ORCH":          {domain.InSection("FLOOR"), two, false},
		"6+ and in FLOOR":               {domain.And(domain.SeatsAtLeast(6), domain.InSection("FLOOR")), six, true},
		"not in FLOOR, given ORCH":      {domain.Not(domain.InSection("FLOOR")), two, true},
		"6+ or in ORCH, given 2 ORCH":   {domain.Or(domain.SeatsAtLeast(6), domain.InSection("ORCH")), two, true},
		"6+ and in ORCH, given 6 FLOOR": {domain.And(domain.SeatsAtLeast(6), domain.InSection("ORCH")), six, false},
	} {
		if got := tt.spec.IsSatisfiedBy(tt.ctx); got != tt.want {
			t.Errorf("%s: IsSatisfiedBy = %v, want %v", name, got, tt.want)
		}
	}
}

// TKT-15: total = subtotal + service fee + VAT on both. The fee is the first
// matching rule's percentage, never below its minimum; VAT is the venue
// country's rate.
func TestPricingPolicy(t *testing.T) {
	policy := domain.PricingPolicy{
		Fees: []domain.FeeRule{
			{Name: "group", When: domain.SeatsAtLeast(6), Percent: 700, Minimum: eur(t, 150)},
			{Name: "standard", When: domain.AnyOrder(), Percent: 1000, Minimum: eur(t, 150)},
		},
		VAT: map[string]int64{"PT": 600, "ES": 2100},
	}

	for name, tt := range map[string]struct {
		lines                  []domain.OrderLine
		country                string
		subtotal, fee, vat, to int64
	}{
		"2 ORCH in Portugal: 10% fee, 6% VAT": {lines(t, "ORCH/A/1", "ORCH/A/2"), "PT", 9000, 900, 594, 10494},
		"1 FLOOR: the fee's minimum":          {lines(t, "FLOOR/GA/0001"), "PT", 2500, 250, 165, 2915},
		"cheap: minimum EUR 1.50":             {[]domain.OrderLine{{Seat: refs(t, "FLOOR/GA/0001")[0], Price: eur(t, 1000)}}, "PT", 1000, 150, 69, 1219},
		"6 FLOOR: the group rule first": {lines(t, "FLOOR/GA/0001", "FLOOR/GA/0002", "FLOOR/GA/0003",
			"FLOOR/GA/0004", "FLOOR/GA/0005", "FLOOR/GA/0006"), "ES", 15000, 1050, 3371, 19421},
	} {
		got, err := policy.Price(tt.lines, tt.country)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want := domain.PriceBreakdown{Subtotal: eur(t, tt.subtotal), Fee: eur(t, tt.fee), VAT: eur(t, tt.vat), Total: eur(t, tt.to)}
		if got != want {
			t.Errorf("%s: %v, want %v", name, got, want)
		}
	}

	if _, err := policy.Price(lines(t, "ORCH/A/1"), "XX"); !errors.Is(err, domain.ErrNoVATRate) {
		t.Errorf("unknown country: error = %v, want %v", err, domain.ErrNoVATRate)
	}
}
