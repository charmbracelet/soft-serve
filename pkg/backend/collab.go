package backend

import (
	"context"
	"errors"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/charmbracelet/soft-serve/pkg/webhook"
)

// AddCollaborator adds a collaborator to a repository.
//
// It implements backend.Backend.
func (d *Backend) AddCollaborator(ctx context.Context, repo string, username string, level access.AccessLevel) error {
	username = strings.ToLower(username)
	if err := utils.ValidateHandle(username); err != nil {
		return err
	}

	repo = utils.SanitizeRepo(repo)
	r, err := d.Repository(ctx, repo)
	if err != nil {
		return err
	}

	if err := db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.AddCollabByUsernameAndRepo(ctx, tx, username, repo, level)
		}),
	); err != nil {
		return err
	}

	wh, err := webhook.NewCollaboratorEvent(ctx, proto.UserFromContext(ctx), r, username, webhook.CollaboratorEventAdded)
	if err != nil {
		return err
	}

	return webhook.SendEvent(ctx, wh)
}

// Collaborators returns a list of collaborators for a repository.
//
// It implements backend.Backend.
func (d *Backend) Collaborators(ctx context.Context, repo string) ([]string, error) {
	repo = utils.SanitizeRepo(repo)
	var users []models.User
	var usernames []string
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		users, err = d.store.ListCollabsByRepoAsUsers(ctx, tx, repo)
		if err != nil {
			return err
		}

		ids := make([]int64, len(users))
		for i, u := range users {
			ids[i] = u.ID
		}

		handles, err := d.store.ListHandlesForIDs(ctx, tx, ids)
		if err != nil {
			return err
		}

		for _, h := range handles {
			usernames = append(usernames, h.Handle)
		}

		return nil
	}); err != nil {
		return nil, db.WrapError(err)
	}

	return usernames, nil
}

// IsCollaborator returns the access level and true if the user is a collaborator of the repository.
//
// It implements backend.Backend.
func (d *Backend) IsCollaborator(ctx context.Context, repo string, username string) (access.AccessLevel, bool, error) {
	if username == "" {
		return -1, false, nil
	}

	repo = utils.SanitizeRepo(repo)
	var m models.Collab
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		m, err = d.store.GetCollabByUsernameAndRepo(ctx, tx, username, repo)
		return err
	}); err != nil {
		return -1, false, db.WrapError(err)
	}

	return m.AccessLevel, m.ID > 0, nil
}

// RemoveCollaborator removes a collaborator from a repository.
//
// It implements backend.Backend.
func (d *Backend) RemoveCollaborator(ctx context.Context, repo string, username string) error {
	repo = utils.SanitizeRepo(repo)
	r, err := d.Repository(ctx, repo)
	if err != nil {
		return err
	}

	wh, err := webhook.NewCollaboratorEvent(ctx, proto.UserFromContext(ctx), r, username, webhook.CollaboratorEventRemoved)
	if err != nil {
		return err
	}

	if err := db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.RemoveCollabByUsernameAndRepo(ctx, tx, username, repo)
		}),
	); err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			return proto.ErrCollaboratorNotFound
		}

		return err
	}

	return webhook.SendEvent(ctx, wh)
}
