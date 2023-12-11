package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// TeamStore is a store for teams.
type TeamStore interface {
	CreateTeam(ctx context.Context, h db.Handler, user, org int64, name string) (models.Team, error)
	ListTeams(ctx context.Context, h db.Handler, user int64) ([]models.Team, error)
	GetTeamByID(ctx context.Context, h db.Handler, user, id int64) (models.Team, error)
	FindTeamByOrgName(ctx context.Context, h db.Handler, user, org int64, name string) (models.Team, error)
	FindTeamByName(ctx context.Context, h db.Handler, user int64, name string) ([]models.Team, error)
	DeleteTeamByID(ctx context.Context, h db.Handler, id int64) error
	AddUserToTeam(ctx context.Context, h db.Handler, team, user int64, lvl access.AccessLevel) error
	RemoveUserFromTeam(ctx context.Context, h db.Handler, team, user int64) error
	UpdateUserAccessInTeam(ctx context.Context, h db.Handler, team, user int64, lvl access.AccessLevel) error
}
