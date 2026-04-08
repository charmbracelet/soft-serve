package git

import "strings"

const maxDiffSize = 512 * 1024 // 512 KB

// DiffRefs returns the unified diff between two git refs (branches, tags, commits).
// Returns empty string if refs are identical or if either ref is empty.
// Output is capped at 512 KB to prevent unbounded memory use.
func (r *Repository) DiffRefs(from, to string) (string, error) {
	if from == "" || to == "" {
		return "", nil
	}
	// Use git diff --no-color from..to to get unified diff between two refs
	out, err := NewCommand("diff", "--no-color", from+".."+to).RunInDir(r.Path)
	if err != nil {
		// git diff exits non-zero only on errors (e.g. invalid ref, not a git repo).
		// Non-zero exit with output means partial results; return them.
		if len(out) > 0 {
			return truncateDiff(string(out)), nil
		}
		return "", err
	}
	return truncateDiff(string(out)), nil
}

// truncateDiff trims trailing newlines and caps the output at maxDiffSize.
func truncateDiff(s string) string {
	if len(s) > maxDiffSize {
		return s[:maxDiffSize] + "\n\n[diff truncated — output exceeded 512 KB]"
	}
	return strings.TrimRight(s, "\n")
}
