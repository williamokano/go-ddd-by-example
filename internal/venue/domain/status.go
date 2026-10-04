package domain

// Status is where a venue is in its lifecycle: Draft → Active → Retired.
type Status uint8

// The venue lifecycle.
const (
	Draft Status = iota + 1
	Active
	Retired
)

// String returns the lower-case name of the status. These strings are what
// gets persisted, so they are part of the storage format.
func (s Status) String() string {
	switch s {
	case Draft:
		return "draft"
	case Active:
		return "active"
	case Retired:
		return "retired"
	default:
		return "unknown"
	}
}
