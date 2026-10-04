# Fees and taxes (9.8)

**Where does this logic go?** It doesn't go in `Order`: the order shouldn't
know tax tables or Stagehand's commercial terms. It doesn't go in
`SectionInventory`, which sells seats at face value. And it doesn't go in the
`CheckoutHandler`, because then the business rules would live in the
application layer. It is a **domain service**, `domain.PricingPolicy`: a
pure function of the lines, the fee rules and the venue's country, which
returns a `PriceBreakdown`. `PlaceOrder` asks it, and the Order records the
answer, so the price can't change after the fact when the rules do.

The fee rules use the **specification pattern**. `SeatsAtLeast(6)`,
`InSection("FLOOR")`, `And`, `Or` and `Not` are values, so "7% for groups of
six or more in the stalls" is `And(SeatsAtLeast(6), InSection("ORCH"))`
added to a slice, not a new `if`. The first matching rule wins, and its
minimum always applies. `StandardPricing()` holds Stagehand's current terms;
the composition root picks it.

What it took to get the venue's country to Ticketing: two **additive** fields,
`country` on `venue.activated.v1` and `venue_country` on `show.published.v2`.
They kept their versions, because old consumers ignore the new field (9.3's
lesson again). Existing rows default to `PT`, where every venue so far is.

TKT-15 in numbers (`TestPricingPolicy`): 2 × EUR 45 in Portugal is EUR 90.00
of seats, plus a EUR 9.00 fee (10%), plus EUR 5.94 VAT (6% of 99.00), for a
total of EUR 104.94. Money stays in integer cents, and percentages are basis
points rounded half up (`Money.Percent`).
