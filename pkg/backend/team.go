package backend

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
)

// CreateTeam creates a new team for an organization.
func (d *Backend) CreateTeam(ctx context.Context, org proto.Org, owner proto.User, name string) (proto.Team, error) {
	m, err := d.store.CreateTeam(ctx, d.db, owner.ID(), org.ID(), name)
	if err != nil {
		return team{}, err
	}
	return team{m}, err
}

// ListTeams lists all teams for a user.
func (d *Backend) ListTeams(ctx context.Context, user proto.User) ([]proto.Team, error) {
	teams, err := d.store.ListTeams(ctx, d.db, user.ID())
	var r []proto.Team
	for _, m := range teams {
		r = append(r, team{m})
	}
	return r, err
}

// GetTeam gets a team by organization id and team name.
func (d *Backend) GetTeam(ctx context.Context, user proto.User, org proto.Org, name string) (proto.Team, error) {
	m, err := d.store.FindTeamByOrgName(ctx, d.db, user.ID(), org.ID(), name)
	if err != nil {
		return team{}, err
	}
	return team{m}, err
}

// FindTeam finds a team by name.
func (d *Backend) FindTeam(ctx context.Context, user proto.User, name string) ([]proto.Team, error) {
	m, err := d.store.FindTeamByName(ctx, d.db, user.ID(), name)
	var r []proto.Team
	for _, m := range m {
		r = append(r, team{m})
	}
	return r, err
}

// DeleteTeam deletes a team.
func (d *Backend) DeleteTeam(ctx context.Context, _ proto.User, team proto.Team) error {
	return d.store.DeleteTeamByID(ctx, d.db, team.ID())
}

type team struct {
	t models.Team
}

var _ proto.Team = team{}

// ID implements proto.Team.
func (t team) ID() int64 {
	return t.t.ID
}

// Name implements proto.Team.
func (t team) Name() string {
	return t.t.Name
}

// Org implements proto.Team.
func (t team) Org() int64 {
	return t.t.OrganizationID
}
