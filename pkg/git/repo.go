package git

import (
	"github.com/gogs/git-module"
)

// Repository is a wrapper around git.Repository with helper methods.
type Repository struct {
	*git.Repository
	Path string
}

// Open opens a git repository at the given path.
func Open(path string) (*Repository, error) {
	repo, err := git.Open(path)
	if err != nil {
		return nil, err
	}
	return &Repository{
		Repository: repo,
		Path:       path,
	}, nil
}

// HEAD returns the HEAD reference for a repository.
func (r *Repository) HEAD() (*Reference, error) {
	rn, err := r.SymbolicRef()
	if err != nil {
		return nil, err
	}
	hash, err := r.ShowRefVerify(rn)
	if err != nil {
		return nil, err
	}
	return &Reference{
		Reference: &git.Reference{
			ID:      hash,
			Refspec: rn,
		},
		Hash: Hash(hash),
		path: r.Path,
	}, nil
}

// References returns the references for a repository.
func (r *Repository) References() ([]*Reference, error) {
	refs, err := r.ShowRef()
	if err != nil {
		return nil, err
	}
	rrefs := make([]*Reference, 0, len(refs))
	for _, ref := range refs {
		rrefs = append(rrefs, &Reference{
			Reference: ref,
			Hash:      Hash(ref.ID),
			path:      r.Path,
		})
	}
	return rrefs, nil
}

// Tree returns the tree for the given reference.
func (r *Repository) Tree(ref *Reference) (*Tree, error) {
	if ref == nil {
		rref, err := r.HEAD()
		if err != nil {
			return nil, err
		}
		ref = rref
	}
	tree, err := r.LsTree(ref.Hash.String())
	if err != nil {
		return nil, err
	}
	return &Tree{
		Tree: tree,
		Path: "",
	}, nil
}

// Patch returns the patch for the given reference.
func (r *Repository) Patch(commit *Commit) (string, error) {
	ddiff, err := r.Diff(commit.Hash.String(), 1000, 1000, 1000)
	if err != nil {
		return "", err
	}
	files := make([]*DiffFile, 0, len(ddiff.Files))
	for _, df := range ddiff.Files {
		sections := make([]*DiffSection, 0, len(df.Sections))
		for _, ds := range df.Sections {
			sections = append(sections, &DiffSection{
				DiffSection: ds,
			})
		}
		files = append(files, &DiffFile{
			DiffFile: df,
			Sections: sections,
		})
	}
	diff := &Diff{
		Diff:  ddiff,
		Files: files,
	}
	return diff.Patch(), err
}
