// Package domain is the Venue Management domain model: venues, their sections
// and rows, the lifecycle Draft → Active → Retired, and the rules VEN-1…VEN-7.
//
// It is pure business logic. It imports only the standard library and
// github.com/google/uuid: no context, no database, no HTTP, no logging, and
// never time.Now(). Time and new IDs are passed in by the application layer.
package domain
