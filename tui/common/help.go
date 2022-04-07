package common

type BubbleHelper interface {
	Help() []HelpEntry
}

type HelpEntry struct {
	Key   string
	Value string
}
