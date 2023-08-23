package webhook

import (
	"context"
	"fmt"

	"github.com/charmbracelet/soft-serve/git"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/charmbracelet/soft-serve/server/store"
)

// PushEvent is a push event.
type PushEvent struct {
	Common

	// Ref is the branch or tag name.
	Ref string `json:"ref" url:"ref"`
	// Before is the previous commit SHA.
	Before string `json:"before" url:"before"`
	// After is the current commit SHA.
	After string `json:"after" url:"after"`
	// Commits is the list of commits.
	Commits []Commit `json:"commits" url:"commits"`
}

// NewPushEvent sends a push event.
func NewPushEvent(ctx context.Context, user proto.User, repo proto.Repository, ref, before, after string) (PushEvent, error) {
	event := EventPush

	payload := PushEvent{
		Ref:    ref,
		Before: before,
		After:  after,
		Common: Common{
			EventType: event,
			Repository: Repository{
				ID:          repo.ID(),
				Name:        repo.Name(),
				Description: repo.Description(),
				ProjectName: repo.ProjectName(),
				Private:     repo.IsPrivate(),
				CreatedAt:   repo.CreatedAt(),
				UpdatedAt:   repo.UpdatedAt(),
			},
			Sender: User{
				ID:       user.ID(),
				Username: user.Username(),
			},
		},
	}

	cfg := config.FromContext(ctx)
	payload.Repository.HTMLURL = repoURL(cfg.HTTP.PublicURL, repo.Name())
	payload.Repository.SSHURL = repoURL(cfg.SSH.PublicURL, repo.Name())
	payload.Repository.GitURL = repoURL(cfg.Git.PublicURL, repo.Name())

	// Find repo owner.
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	owner, err := datastore.GetUserByID(ctx, dbx, repo.UserID())
	if err != nil {
		return PushEvent{}, db.WrapError(err)
	}

	payload.Repository.Owner.ID = owner.ID
	payload.Repository.Owner.Username = owner.Username

	// Find commits.
	r, err := repo.Open()
	if err != nil {
		return PushEvent{}, err
	}

	payload.Repository.DefaultBranch, err = proto.RepositoryDefaultBranch(repo)
	if err != nil {
		return PushEvent{}, err
	}

	rev := after
	if !git.IsZeroHash(before) {
		rev = fmt.Sprintf("%s..%s", before, after)
	}

	commits, err := r.Log(rev)
	if err != nil {
		return PushEvent{}, err
	}

	payload.Commits = make([]Commit, len(commits))
	for i, c := range commits {
		payload.Commits[i] = Commit{
			ID:      c.ID.String(),
			Message: c.Message,
			Title:   c.Summary(),
			Author: Author{
				Name:  c.Author.Name,
				Email: c.Author.Email,
				Date:  c.Author.When,
			},
			Committer: Author{
				Name:  c.Committer.Name,
				Email: c.Committer.Email,
				Date:  c.Committer.When,
			},
			Timestamp: c.Committer.When,
		}
	}

	return payload, nil
}
