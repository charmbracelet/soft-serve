package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/proto"
	"github.com/charmbracelet/soft-serve/server/db/types"
)

var _ proto.Provider = &Config{}
var _ proto.MetadataProvider = &Config{}

// Metadata returns the repository's metadata.
func (c *Config) Metadata(name string) (proto.Metadata, error) {
	i, err := c.db.GetRepo(name)
	if err != nil {
		return nil, err
	}
	return &repo{
		cfg:  c,
		info: i,
	}, nil
}

// Open opens a repository.
func (c *Config) Open(name string) (proto.Repository, error) {
	if name == "" {
		return nil, os.ErrNotExist
	}
	r, err := git.Open(filepath.Join(c.RepoPath(), name+".git"))
	if err != nil {
		log.Printf("error opening repository %q: %v", name, err)
		return nil, err
	}
	return &repo{
		cfg:  c,
		repo: r,
	}, nil
}

// ListRepos lists all repositories metadata.
func (c *Config) ListRepos() ([]proto.Metadata, error) {
	md := make([]proto.Metadata, 0)
	ds, err := os.ReadDir(c.RepoPath())
	if err != nil {
		return nil, err
	}
	for _, d := range ds {
		name := strings.TrimSuffix(d.Name(), ".git")
		r, err := c.db.GetRepo(name)
		if err != nil || r == nil {
			md = append(md, &emptyMetadata{
				name: name,
				cfg:  c,
			})
		} else {
			md = append(md, &repo{
				cfg:  c,
				info: r,
			})
		}
	}
	return md, nil
}

var _ proto.Metadata = emptyMetadata{}

type emptyMetadata struct {
	name string
	cfg  *Config
}

// Collabs implements proto.Metadata.
func (emptyMetadata) Collabs() []proto.User {
	return []proto.User{}
}

// Description implements proto.Metadata.
func (emptyMetadata) Description() string {
	return ""
}

// IsPrivate implements proto.Metadata.
func (emptyMetadata) IsPrivate() bool {
	return false
}

// Name implements proto.Metadata.
func (e emptyMetadata) Name() string {
	return e.name
}

// Open implements proto.Metadata.
func (e emptyMetadata) Open() (proto.Repository, error) {
	return e.cfg.Open(e.Name())
}

// ProjectName implements proto.Metadata.
func (emptyMetadata) ProjectName() string {
	return ""
}

var _ proto.Metadata = &repo{}
var _ proto.Repository = &repo{}

// repo represents a Git repository.
type repo struct {
	cfg  *Config
	repo *git.Repository
	info *types.Repo
}

// Open opens the underlying Repository.
func (r *repo) Open() (proto.Repository, error) {
	return r.cfg.Open(r.Name())
}

// Name returns the name of the repository.
func (r *repo) Name() string {
	if r.repo != nil {
		strings.TrimSuffix(filepath.Base(r.repo.Path), ".git")
	}
	return r.info.Name
}

// ProjectName returns the repository's project name.
func (r *repo) ProjectName() string {
	return r.info.ProjectName
}

// Description returns the repository's description.
func (r *repo) Description() string {
	return r.info.Description
}

// IsPrivate returns true if the repository is private.
func (r *repo) IsPrivate() bool {
	return r.info.Private
}

// Collabs returns the repository's collaborators.
func (r *repo) Collabs() []proto.User {
	collabs := make([]proto.User, 0)
	cs, err := r.cfg.db.ListRepoCollabs(r.Name())
	if err != nil {
		return collabs
	}
	for i, c := range cs {
		ks, err := r.cfg.db.GetUserPublicKeys(c)
		if err != nil {
			log.Printf("error getting public keys for %q: %v", c.Name, err)
			continue
		}
		u := &user{
			user: c,
			keys: ks,
		}
		collabs[i] = u
	}
	return collabs
}

// Repository returns the underlying git.Repository.
func (r *repo) Repository() *git.Repository {
	return r.repo
}
