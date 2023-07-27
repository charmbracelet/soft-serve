package backend

import (
	"context"
	"strings"

	"github.com/charmbracelet/soft-serve/server/db"
	"github.com/charmbracelet/soft-serve/server/db/models"
	"github.com/charmbracelet/soft-serve/server/utils"
)

// AddCollaborator adds a collaborator to a repository.
//
// It implements backend.Backend.
func (d *Backend) AddCollaborator(ctx context.Context, repo string, username string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateUsername(username); err != nil {
		return err
	}

	repo = utils.SanitizeRepo(repo)
	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.AddCollabByUsernameAndRepo(ctx, tx, username, repo)
		}),
	)
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

// IsCollaborator returns true if the user is a collaborator of the repository.
//
// It implements backend.Backend.
func (d *Backend) IsCollaborator(ctx context.Context, repo string, username string) (bool, error) {
	if username == "" {
		return false, nil
	}

	repo = utils.SanitizeRepo(repo)
	var m models.Collab
	if err := d.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		m, err = d.store.GetCollabByUsernameAndRepo(ctx, tx, username, repo)
		return err
	}); err != nil {
		return false, db.WrapError(err)
	}

	return m.ID > 0, nil
}

// RemoveCollaborator removes a collaborator from a repository.
//
// It implements backend.Backend.
func (d *Backend) RemoveCollaborator(ctx context.Context, repo string, username string) error {
	repo = utils.SanitizeRepo(repo)
	return db.WrapError(
		d.db.TransactionContext(ctx, func(tx *db.Tx) error {
			return d.store.RemoveCollabByUsernameAndRepo(ctx, tx, username, repo)
		}),
	)
}
