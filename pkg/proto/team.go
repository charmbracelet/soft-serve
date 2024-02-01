package proto

// Team is an interface representing a team.
type Team interface {
	// ID returns the user's ID.
	ID() int64
	// Name returns the org's name.
	Name() string
	// Parent organization's ID.
	Org() int64
}
