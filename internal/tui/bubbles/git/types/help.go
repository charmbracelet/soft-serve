package types

type HelpableBubble interface {
	Help() []HelpEntry
}

type HelpEntry struct {
	Key   string
	Value string
}
