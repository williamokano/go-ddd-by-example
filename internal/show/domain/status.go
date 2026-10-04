package domain

// Status is where a show is in its lifecycle (Chapter 1's state machine).
type Status uint8

// The show lifecycle.
const (
	Draft Status = iota + 1
	Published
	SoldOut
	Cancelled
	Completed
)

var statusNames = map[Status]string{
	Draft: "draft", Published: "published", SoldOut: "sold_out", Cancelled: "cancelled", Completed: "completed",
}

// String returns the stored name of the status.
func (s Status) String() string {
	if name, ok := statusNames[s]; ok {
		return name
	}
	return "unknown"
}

// IsTerminal reports whether nothing can happen to the show any more (SHW-6).
func (s Status) IsTerminal() bool { return s == Cancelled || s == Completed }
