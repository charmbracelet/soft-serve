package git

import "strings"

// DiffRefs returns the unified diff between two git refs (branches, tags, commits).
// Returns empty string if refs are identical or if either ref is empty.
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
			return strings.TrimRight(string(out), "\n"), nil
		}
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}
