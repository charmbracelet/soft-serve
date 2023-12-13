package backend

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
)

// CreateOrg creates a new organization.
func (d *Backend) CreateOrg(ctx context.Context, owner proto.User, name, email string) (proto.Org, error) {
	o, err := d.store.CreateOrg(ctx, d.db, owner.ID(), name, email)
	if err != nil {
		return org{}, err
	}
	return org{o}, err
}

// ListOrgs lists all organizations for a user.
func (d *Backend) ListOrgs(ctx context.Context, user proto.User) ([]proto.Org, error) {
	orgs, err := d.store.ListOrgs(ctx, d.db, user.ID())
	var r []proto.Org
	for _, o := range orgs {
		r = append(r, org{o})
	}
	return r, err
}

// FindOrganization finds an organization belonging to a user by name.
func (d *Backend) FindOrganization(ctx context.Context, user proto.User, name string) (proto.Org, error) {
	o, err := d.store.FindOrgByHandle(ctx, d.db, user.ID(), name)
	return org{o}, err
}

// DeleteOrganization deletes an organization for a user.
func (d *Backend) DeleteOrganization(ctx context.Context, user proto.User, name string) error {
	o, err := d.store.FindOrgByHandle(ctx, d.db, user.ID(), name)
	if err != nil {
		return err
	}
	return d.store.DeleteOrgByID(ctx, d.db, user.ID(), o.ID)
}

type org struct {
	o models.Organization
}

var _ proto.Org = org{}

// DisplayName implements proto.Org.
func (o org) DisplayName() string {
	if o.o.Name.Valid {
		return o.o.Name.String
	}
	return ""
}

// ID implements proto.Org.
func (o org) ID() int64 {
	return o.o.ID
}

// Handle implements proto.Org.
func (o org) Handle() string {
	return o.o.Handle.Handle
}
