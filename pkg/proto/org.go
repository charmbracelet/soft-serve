package proto

// Org is an interface representing a organization.
type Org interface {
	// ID returns the user's ID.
	ID() int64
	// Handle returns the org's name.
	Handle() string
	// DisplayName returns the org's display name.
	DisplayName() string
}
