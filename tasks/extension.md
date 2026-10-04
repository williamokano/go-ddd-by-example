# Extension track progress (optional)

Separate from `tasks/todo.md`: the main course does not depend on this track. Pages are
`docs/ext-*.html`; decisions are in `docs/ext-decisions.html`. Prerequisites per extension are in
`docs/ext-0-overview.html`.

**Current lesson:** not started

Order: E1 any time after Part 7 · E2 → E4 (audit needs the actor) · E3 any time after Part 5 ·
E2 → E5 → E6.

## E1 — Pricing (`docs/ext-1-pricing.html`)
- [ ] E1.1 Money you can trust: rates, rounding and allocation
- [ ] E1.2 The PriceBreakdown value object
- [ ] E1.3 Demand pricing as a pure policy
- [ ] E1.4 The price lock: teach ShowInventory to quote
- [ ] E1.5 Discounts: specifications and stacking
- [ ] E1.6 Fees
- [ ] E1.7 Taxes, and the one fact no contract carries
- [ ] E1.8 Put it together: OrderPricing, checkout and a preview
- [ ] E1.9 Stretch: fees as a domain service
- [ ] E1.10 Stretch: redemption limits

## E2 — Identity & access (`docs/ext-2-identity.html`)
- [ ] E2.1 Principal, permissions and roles
- [ ] E2.2 The User aggregate and local credentials
- [ ] E2.3 Login use cases and the IdentityProvider port
- [ ] E2.4 Persistence and the token adapter
- [ ] E2.5 The authentication middleware
- [ ] E2.6 RBAC as a decorator
- [ ] E2.7 Ownership: VEN-8
- [ ] E2.8 A second source: OIDC SSO
- [ ] E2.9 Architecture test and threat checklist

## E3 — Observability (`docs/ext-3-observability.html`)
- [ ] E3.1 Metrics endpoint and HTTP RED
- [ ] E3.2 Use-case decorator
- [ ] E3.3 Business metrics from events and outcomes
- [ ] E3.4 Infrastructure metrics
- [ ] E3.5 Prometheus + Grafana Compose profile
- [ ] E3.6 Stretch: SLO and alert

## E4 — Audit trail (`docs/ext-4-audit.html`)
- [ ] E4.1 Decide what belongs in the trail
- [ ] E4.2 Entry, port, append-only table
- [ ] E4.3 Command-level audit decorator
- [ ] E4.4 Actor in event metadata
- [ ] E4.5 Protected query API
- [ ] E4.6 Stretch: tamper evidence

## E5 — Payments (`docs/ext-5-payments.html`)
- [ ] E5.1 Event-storm and the Payment state machine
- [ ] E5.2 Refunds inside the aggregate
- [ ] E5.3 Provider port, notifications, scriptable fake
- [ ] E5.4 Use cases: idempotency and unknown outcomes
- [ ] E5.5 Persistence, outbox, published language
- [ ] E5.6 Rewire Ticketing: authorize, confirm, capture
- [ ] E5.7 The Stripe adapter (sandbox)
- [ ] E5.8 Webhooks
- [ ] E5.9 Feature: customer refunds
- [ ] E5.10 Feature: chargebacks, accepted on arrival
- [ ] E5.11 The Denylist context
- [ ] E5.12 Block on chargeback, enforce at purchase
- [ ] E5.13 Reconciliation and failure drills
- [ ] E5.14 Stretch: defend disputes

## E6 — Fraud screening (`docs/ext-6-fraud.html`)
- [ ] E6.1 Model first
- [ ] E6.2 RiskAssessment and RiskPolicy
- [ ] E6.3 Screener and OutcomeReporter ports, and a fake
- [ ] E6.4 ScreenOrder: timeouts and circuit breaker
- [ ] E6.5 New saga step
- [ ] E6.6 Asynchronous manual review
- [ ] E6.7 Outcome feedback
- [ ] E6.8 Chargebacks and pre-dispute alerts
- [ ] E6.9 Failure drills and retrospective

## Notes

(Filled in as extensions are completed: what went well, what was hard, decisions changed.)
