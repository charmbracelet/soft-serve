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

// BranchTagEvent is a branch or tag event.
type BranchTagEvent struct {
	Common

	// Ref is the branch or tag name.
	Ref string `json:"ref" url:"ref"`
	// Before is the previous commit SHA.
	Before string `json:"before" url:"before"`
	// After is the current commit SHA.
	After string `json:"after" url:"after"`
	// Created is whether the branch or tag was created.
	Created bool `json:"created" url:"created"`
	// Deleted is whether the branch or tag was deleted.
	Deleted bool `json:"deleted" url:"deleted"`
}

// NewBranchTagEvent sends a branch or tag event.
func NewBranchTagEvent(ctx context.Context, user proto.User, repo proto.Repository, ref, before, after string) (BranchTagEvent, error) {
	var event Event
	if git.IsZeroHash(before) {
		event = EventBranchTagCreate
	} else if git.IsZeroHash(after) {
		event = EventBranchTagDelete
	} else {
		return BranchTagEvent{}, fmt.Errorf("invalid branch or tag event: before=%q after=%q", before, after)
	}

	payload := BranchTagEvent{
		Ref:     ref,
		Before:  before,
		After:   after,
		Created: git.IsZeroHash(before),
		Deleted: git.IsZeroHash(after),
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
		return BranchTagEvent{}, db.WrapError(err)
	}

	payload.Repository.Owner.ID = owner.ID
	payload.Repository.Owner.Username = owner.Username
	payload.Repository.DefaultBranch, err = proto.RepositoryDefaultBranch(repo)
	if err != nil {
		return BranchTagEvent{}, err
	}

	return payload, nil
}
