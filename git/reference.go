package git

import (
	"strings"

	"github.com/gogs/git-module"
)

const (
	// HEAD represents the name of the HEAD reference.
	HEAD = "HEAD"
	// RefsHeads represents the prefix for branch references.
	RefsHeads = git.RefsHeads
	// RefsTags represents the prefix for tag references.
	RefsTags = git.RefsTags
)

// Reference is a wrapper around git.Reference with helper methods.
type Reference struct {
	*git.Reference
	path string // repo path
}

// ReferenceName is a Refspec wrapper.
type ReferenceName string

// String returns the reference name i.e. refs/heads/master.
func (r ReferenceName) String() string {
	return string(r)
}

// Short returns the short name of the reference i.e. master.
func (r ReferenceName) Short() string {
	return git.RefShortName(string(r))
}

// Name returns the reference name i.e. refs/heads/master.
func (r *Reference) Name() ReferenceName {
	return ReferenceName(r.Refspec)
}

// IsBranch returns true if the reference is a branch.
func (r *Reference) IsBranch() bool {
	return strings.HasPrefix(r.Refspec, git.RefsHeads)
}

// IsTag returns true if the reference is a tag.
func (r *Reference) IsTag() bool {
	return strings.HasPrefix(r.Refspec, git.RefsTags)
}
