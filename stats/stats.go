package stats

// Stats provides an interface that can be used to collect metrics about the server.
type Stats interface {
	Tui()
	Push()
	Fetch()
}
