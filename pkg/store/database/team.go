package database

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

// TODO: should we return all the org's teams if the user is an org admin?
//       if so, need to join organization_members too on the selects below.

var _ store.TeamStore = (*teamStore)(nil)

type teamStore struct{}

// RemoveUserFromTeam implements store.TeamStore.
func (*teamStore) RemoveUserFromTeam(ctx context.Context, h db.Handler, team int64, user int64) error {
	// TODO: caller perms
	query := h.Rebind(`
		DELETE FROM team_members
		WHERE
		  team_id = ?
		  AND user_id = ?
	`)
	_, err := h.ExecContext(ctx, query, team, user)
	return err
}

// UpdateUserAccessInTeam implements store.TeamStore.
func (*teamStore) UpdateUserAccessInTeam(ctx context.Context, h db.Handler, team int64, user int64, lvl access.AccessLevel) error {
	// TODO: caller perms
	query := h.Rebind(`
		UPDATE team_members
		WHERE
		  team_id = ?
		  AND user_id = ?
		SET
		  access_level = ?
	`)
	_, err := h.ExecContext(ctx, query, team, user, lvl)
	return err
}

// AddUserToTeam implements store.TeamStore.
func (*teamStore) AddUserToTeam(ctx context.Context, h db.Handler, team int64, user int64, lvl access.AccessLevel) error {
	// TODO: caller perms
	query := h.Rebind(`
		INSERT INTO
		  team_members (team_id, user_id, access_level, updated_at)
		VALUES
		  (?, ?, ?, CURRENT_TIMESTAMP);
	`)
	_, err := h.ExecContext(ctx, query, team, user, lvl)
	return err
}

// CreateTeam implements store.TeamStore.
func (s *teamStore) CreateTeam(ctx context.Context, h db.Handler, user, org int64, name string) (models.Team, error) {
	// TODO: caller perms
	// TODO: what the access_level column does on team?
	query := h.Rebind(`
		INSERT INTO
		  teams (organization_id, name)
		VALUES
		  (?, ?) RETURNING *
	`)
	var team models.Team
	if err := h.GetContext(ctx, &team, query, org, name); err != nil {
		return models.Team{}, err
	}
	return team, s.AddUserToTeam(ctx, h, team.ID, user, access.AdminAccess)
}

// DeleteTeamByID implements store.TeamStore.
func (*teamStore) DeleteTeamByID(ctx context.Context, h db.Handler, id int64) error {
	// TODO: caller perms
	query := h.Rebind(`
		DELETE FROM teams
		WHERE
		  id = ?
	`)
	_, err := h.ExecContext(ctx, query, id)
	return err
}

// FindTeamByName implements store.TeamStore.
func (*teamStore) FindTeamByName(ctx context.Context, h db.Handler, uid int64, name string) ([]models.Team, error) {
	query := h.Rebind(`
		SELECT
		  t.*
		FROM
		  teams t
		  JOIN team_members tm ON tm.team_id = t.id
		WHERE
		  tm.user_id = ?
		  AND t.name = ?
	`)
	var teams []models.Team
	err := h.SelectContext(ctx, &teams, query, uid, name)
	return teams, err
}

// FindTeamByOrgName implements store.TeamStore.
func (*teamStore) FindTeamByOrgName(ctx context.Context, h db.Handler, user int64, org int64, name string) (models.Team, error) {
	query := h.Rebind(`
		SELECT
		  t.*
		FROM
		  teams t
		  JOIN team_members tm ON tm.team_id = t.id
		WHERE
		  tm.user_id = ?
		  AND t.organization_id = ?
		  AND t.name = ?
	`)
	var team models.Team
	err := h.GetContext(ctx, &team, query, user, org, name)
	return team, err
}

// GetTeamByID implements store.TeamStore.
func (*teamStore) GetTeamByID(ctx context.Context, h db.Handler, uid, id int64) (models.Team, error) {
	query := h.Rebind(`
		SELECT
		  t.*
		FROM
		  teams t
		  JOIN team_members tm ON tm.team_id = t.id
		WHERE
		  tm.user_id = ?
		  AND t.id = ?
	`)
	var team models.Team
	err := h.GetContext(ctx, &team, query, uid, id)
	return team, err
}

// ListTeams implements store.TeamStore.
func (*teamStore) ListTeams(ctx context.Context, h db.Handler, uid int64) ([]models.Team, error) {
	query := h.Rebind(`
		SELECT
		  t.*
		FROM
		  teams t
		  JOIN team_members tm ON tm.team_id = t.id
		WHERE
		  tm.user_id = ?
	`)
	var teams []models.Team
	err := h.SelectContext(ctx, &teams, query, uid)
	return teams, err
}
