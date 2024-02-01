package database

import (
	"context"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/utils"
)

type collabStore struct{}

var _ store.CollaboratorStore = (*collabStore)(nil)

// AddCollabByUsernameAndRepo implements store.CollaboratorStore.
func (*collabStore) AddCollabByUsernameAndRepo(ctx context.Context, tx db.Handler, username string, repo string, level access.AccessLevel) error {
	username = strings.ToLower(username)
	if err := utils.ValidateHandle(username); err != nil {
		return err
	}

	repo = utils.SanitizeRepo(repo)

	query := tx.Rebind(`INSERT INTO collabs (access_level, user_id, repo_id, updated_at)
			VALUES (
				?,
				(
					SELECT id FROM users WHERE handle_id = (
						SELECT id FROM handles WHERE handle = ?
					)
				),
				(
					SELECT id FROM repos WHERE name = ?
				),
				CURRENT_TIMESTAMP
			);`)
	_, err := tx.ExecContext(ctx, query, level, username, repo)
	return err
}

// GetCollabByUsernameAndRepo implements store.CollaboratorStore.
func (*collabStore) GetCollabByUsernameAndRepo(ctx context.Context, tx db.Handler, username string, repo string) (models.Collab, error) {
	var m models.Collab

	username = strings.ToLower(username)
	if err := utils.ValidateHandle(username); err != nil {
		return models.Collab{}, err
	}

	repo = utils.SanitizeRepo(repo)

	err := tx.GetContext(ctx, &m, tx.Rebind(`
		SELECT
			collabs.*
		FROM
			collabs
		INNER JOIN users ON users.id = collabs.user_id
		INNER JOIN handles ON handles.id = users.handle_id
		INNER JOIN repos ON repos.id = collabs.repo_id
		WHERE
			handles.handle = ? AND repos.name = ?
	`), username, repo)

	return m, err
}

// ListCollabsByRepo implements store.CollaboratorStore.
func (*collabStore) ListCollabsByRepo(ctx context.Context, tx db.Handler, repo string) ([]models.Collab, error) {
	var m []models.Collab

	repo = utils.SanitizeRepo(repo)
	query := tx.Rebind(`
		SELECT
			collabs.*
		FROM
			collabs
		INNER JOIN repos ON repos.id = collabs.repo_id
		WHERE
			repos.name = ?
	`)

	err := tx.SelectContext(ctx, &m, query, repo)
	return m, err
}

// ListCollabsByRepoAsUsers implements store.CollaboratorStore.
func (*collabStore) ListCollabsByRepoAsUsers(ctx context.Context, tx db.Handler, repo string) ([]models.User, error) {
	var m []models.User

	repo = utils.SanitizeRepo(repo)
	query := tx.Rebind(`
		SELECT
			users.*
		FROM
			users
		INNER JOIN collabs ON collabs.user_id = users.id
		INNER JOIN repos ON repos.id = collabs.repo_id
		WHERE
			repos.name = ?
	`)

	err := tx.SelectContext(ctx, &m, query, repo)
	return m, err
}

// RemoveCollabByUsernameAndRepo implements store.CollaboratorStore.
func (*collabStore) RemoveCollabByUsernameAndRepo(ctx context.Context, tx db.Handler, username string, repo string) error {
	username = strings.ToLower(username)
	if err := utils.ValidateHandle(username); err != nil {
		return err
	}

	repo = utils.SanitizeRepo(repo)
	query := tx.Rebind(`
		DELETE FROM
			collabs
		WHERE
			user_id = (
				SELECT id FROM users WHERE handle_id = (
					SELECT id FROM handles WHERE handle = ?
				)
			) AND repo_id = (
				SELECT id FROM repos WHERE name = ?
			)
	`)
	_, err := tx.ExecContext(ctx, query, username, repo)
	return err
}
