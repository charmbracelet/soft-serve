package database

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/utils"
)

type repoStore struct{}

var _ store.RepositoryStore = (*repoStore)(nil)

// CreateRepo implements store.RepositoryStore.
func (*repoStore) CreateRepo(ctx context.Context, tx db.Handler, name string, userID int64, projectName string, description string, isPrivate bool, isHidden bool, isMirror bool) error {
	name = utils.SanitizeRepo(name)
	values := []interface{}{
		name, projectName, description, isPrivate, isMirror, isHidden,
	}
	query := `INSERT INTO repos (name, project_name, description, private, mirror, hidden, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`
	if userID > 0 {
		query = `INSERT INTO repos (name, project_name, description, private, mirror, hidden, updated_at, user_id)
			VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?);`
		values = append(values, userID)
	}

	query = tx.Rebind(query)
	_, err := tx.ExecContext(ctx, query, values...)
	return db.WrapError(err)
}

// DeleteRepoByName implements store.RepositoryStore.
func (*repoStore) DeleteRepoByName(ctx context.Context, tx db.Handler, name string) error {
	name = utils.SanitizeRepo(name)
	query := tx.Rebind("DELETE FROM repos WHERE name = ?;")
	_, err := tx.ExecContext(ctx, query, name)
	return db.WrapError(err)
}

// GetAllRepos implements store.RepositoryStore.
func (*repoStore) GetAllRepos(ctx context.Context, tx db.Handler) ([]models.Repo, error) {
	var repos []models.Repo
	query := tx.Rebind("SELECT * FROM repos;")
	err := tx.SelectContext(ctx, &repos, query)
	return repos, db.WrapError(err)
}

// GetUserRepos implements store.RepositoryStore.
func (*repoStore) GetUserRepos(ctx context.Context, tx db.Handler, userID int64) ([]models.Repo, error) {
	var repos []models.Repo
	query := tx.Rebind("SELECT * FROM repos WHERE user_id = ?;")
	err := tx.SelectContext(ctx, &repos, query, userID)
	return repos, db.WrapError(err)
}

// GetRepoByName implements store.RepositoryStore.
func (*repoStore) GetRepoByName(ctx context.Context, tx db.Handler, name string) (models.Repo, error) {
	var repo models.Repo
	name = utils.SanitizeRepo(name)
	query := tx.Rebind("SELECT * FROM repos WHERE name = ?;")
	err := tx.GetContext(ctx, &repo, query, name)
	return repo, db.WrapError(err)
}

// GetRepoDescriptionByName implements store.RepositoryStore.
func (*repoStore) GetRepoDescriptionByName(ctx context.Context, tx db.Handler, name string) (string, error) {
	var description string
	name = utils.SanitizeRepo(name)
	query := tx.Rebind("SELECT description FROM repos WHERE name = ?;")
	err := tx.GetContext(ctx, &description, query, name)
	return description, db.WrapError(err)
}

// GetRepoIsHiddenByName implements store.RepositoryStore.
func (*repoStore) GetRepoIsHiddenByName(ctx context.Context, tx db.Handler, name string) (bool, error) {
	var isHidden bool
	name = utils.SanitizeRepo(name)
	query := tx.Rebind("SELECT hidden FROM repos WHERE name = ?;")
	err := tx.GetContext(ctx, &isHidden, query, name)
	return isHidden, db.WrapError(err)
}

// GetRepoIsMirrorByName implements store.RepositoryStore.
func (*repoStore) GetRepoIsMirrorByName(ctx context.Context, tx db.Handler, name string) (bool, error) {
	var isMirror bool
	name = utils.SanitizeRepo(name)
	query := tx.Rebind("SELECT mirror FROM repos WHERE name = ?;")
	err := tx.GetContext(ctx, &isMirror, query, name)
	return isMirror, db.WrapError(err)
}

// GetRepoIsPrivateByName implements store.RepositoryStore.
func (*repoStore) GetRepoIsPrivateByName(ctx context.Context, tx db.Handler, name string) (bool, error) {
	var isPrivate bool
	name = utils.SanitizeRepo(name)
	query := tx.Rebind("SELECT private FROM repos WHERE name = ?;")
	err := tx.GetContext(ctx, &isPrivate, query, name)
	return isPrivate, db.WrapError(err)
}

// GetRepoProjectNameByName implements store.RepositoryStore.
func (*repoStore) GetRepoProjectNameByName(ctx context.Context, tx db.Handler, name string) (string, error) {
	var pname string
	name = utils.SanitizeRepo(name)
	query := tx.Rebind("SELECT project_name FROM repos WHERE name = ?;")
	err := tx.GetContext(ctx, &pname, query, name)
	return pname, db.WrapError(err)
}

// SetRepoDescriptionByName implements store.RepositoryStore.
func (*repoStore) SetRepoDescriptionByName(ctx context.Context, tx db.Handler, name string, description string) error {
	name = utils.SanitizeRepo(name)
	query := tx.Rebind("UPDATE repos SET description = ? WHERE name = ?;")
	_, err := tx.ExecContext(ctx, query, description, name)
	return db.WrapError(err)
}

// SetRepoIsHiddenByName implements store.RepositoryStore.
func (*repoStore) SetRepoIsHiddenByName(ctx context.Context, tx db.Handler, name string, isHidden bool) error {
	name = utils.SanitizeRepo(name)
	query := tx.Rebind("UPDATE repos SET hidden = ? WHERE name = ?;")
	_, err := tx.ExecContext(ctx, query, isHidden, name)
	return db.WrapError(err)
}

// SetRepoIsPrivateByName implements store.RepositoryStore.
func (*repoStore) SetRepoIsPrivateByName(ctx context.Context, tx db.Handler, name string, isPrivate bool) error {
	name = utils.SanitizeRepo(name)
	query := tx.Rebind("UPDATE repos SET private = ? WHERE name = ?;")
	_, err := tx.ExecContext(ctx, query, isPrivate, name)
	return db.WrapError(err)
}

// SetRepoNameByName implements store.RepositoryStore.
func (*repoStore) SetRepoNameByName(ctx context.Context, tx db.Handler, name string, newName string) error {
	name = utils.SanitizeRepo(name)
	newName = utils.SanitizeRepo(newName)
	query := tx.Rebind("UPDATE repos SET name = ? WHERE name = ?;")
	_, err := tx.ExecContext(ctx, query, newName, name)
	return db.WrapError(err)
}

// SetRepoProjectNameByName implements store.RepositoryStore.
func (*repoStore) SetRepoProjectNameByName(ctx context.Context, tx db.Handler, name string, projectName string) error {
	name = utils.SanitizeRepo(name)
	query := tx.Rebind("UPDATE repos SET project_name = ? WHERE name = ?;")
	_, err := tx.ExecContext(ctx, query, projectName, name)
	return db.WrapError(err)
}
