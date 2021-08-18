package tui

import "fmt"

type helpEntry struct {
	key string
	val string
}

func (h helpEntry) String() string {
	return fmt.Sprintf("%s %s", helpKeyStyle.Render(h.key), helpValueStyle.Render(h.val))
}
