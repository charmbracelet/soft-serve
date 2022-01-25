package types

import "github.com/muesli/reflow/truncate"

func TruncateString(s string, max int, tail string) string {
	if max < 0 {
		max = 0
	}
	return truncate.StringWithTail(s, uint(max), tail)
}
