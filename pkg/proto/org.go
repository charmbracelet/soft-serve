package proto

// Org is an interface representing a organization.
type Org interface {
	// ID returns the user's ID.
	ID() int64
	// Name returns the org's name.
	Name() string
	// DisplayName
	DisplayName() string
}
