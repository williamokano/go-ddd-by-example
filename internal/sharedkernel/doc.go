// Package sharedkernel is the small piece of model every context shares
// identically (ADR-009): Money and Currency, and the domain-event building
// block. It is domain, not utilities; it imports only the standard library,
// and changing it is a change to every context, so it stays tiny.
package sharedkernel
