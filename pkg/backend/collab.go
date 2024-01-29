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
	if err := utils.ValidateUsername(username); err != nil {
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
		if errors.Is(err, db.ErrDuplicateKey) {
			return proto.ErrCollaboratorExist
		}

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
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		users, err = d.store.ListCollabsByRepoAsUsers(ctx, tx, repo)
		return err
	}); err != nil {
		return nil, db.WrapError(err)
	}

	var usernames []string
	for _, u := range users {
		usernames = append(usernames, u.Username)
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
