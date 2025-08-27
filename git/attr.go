// Package git provides git repository operations and utilities.
package git

import (
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Attribute represents a Git attribute.
type Attribute struct {
	Name  string
	Value string
}

// CheckAttributes checks the attributes of the given ref and path.
func (r *Repository) CheckAttributes(ref *Reference, path string) ([]Attribute, error) {
	rnd := rand.NewSource(time.Now().UnixNano())
	fn := "soft-serve-index-" + strconv.Itoa(rand.New(rnd).Int()) //nolint: gosec
	tmpindex := filepath.Join(os.TempDir(), fn)

	defer os.Remove(tmpindex) //nolint: errcheck

	readTree := NewCommand("read-tree", "--reset", "-i", ref.Name().String()).
		AddEnvs("GIT_INDEX_FILE=" + tmpindex)
	if _, err := readTree.RunInDir(r.Path); err != nil {
		return nil, err //nolint:wrapcheck
	}

	checkAttr := NewCommand("check-attr", "--cached", "-a", "--", path).
		AddEnvs("GIT_INDEX_FILE=" + tmpindex)
	out, err := checkAttr.RunInDir(r.Path)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return parseAttributes(path, out), nil
}

func parseAttributes(path string, buf []byte) []Attribute {
	attrs := make([]Attribute, 0)
	for _, line := range strings.Split(string(buf), "\n") {
		if line == "" {
			continue
		}

		line = strings.TrimPrefix(line, path+": ")
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		attrs = append(attrs, Attribute{
			Name:  parts[0],
			Value: parts[1],
		})
	}

	return attrs
}
