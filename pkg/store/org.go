package store

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
)

// OrgStore is a store for organizations.
type OrgStore interface {
	CreateOrg(ctx context.Context, h db.Handler, user int64, name, email string) (models.Organization, error)
	ListOrgs(ctx context.Context, h db.Handler, user int64) ([]models.Organization, error)
	GetOrgByID(ctx context.Context, h db.Handler, user, id int64) (models.Organization, error)
	FindOrgByHandle(ctx context.Context, h db.Handler, user int64, name string) (models.Organization, error)
	DeleteOrgByID(ctx context.Context, h db.Handler, user, id int64) error
	AddUserToOrg(ctx context.Context, h db.Handler, org, user int64, lvl access.AccessLevel) error
	RemoveUserFromOrg(ctx context.Context, h db.Handler, org, user int64) error
	UpdateUserAccessInOrg(ctx context.Context, h db.Handler, org, user int64, lvl access.AccessLevel) error
	UpdateOrgContactEmail(ctx context.Context, h db.Handler, org int64, email string) error
	// TODO: rename org?
	// XXX: what else?
}
