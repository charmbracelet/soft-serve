package stats

import "log"

// Stats provides an interface that can be used to collect metrics about the server.
type Stats interface {
	Tui(action string)
	Push(repo string)
	Fetch(repo string)
}

type stats struct{}

func (s *stats) Tui(action string) {
	log.Printf("TUI: %s", action)
}

func (s *stats) Push(repo string) {
	log.Printf("git push: %s", repo)
}

func (s *stats) Fetch(repo string) {
	log.Printf("git fetch: %s", repo)
}

func NewStats() Stats {
	return &stats{}
}
