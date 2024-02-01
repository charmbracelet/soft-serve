package database

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/store"
	"github.com/charmbracelet/soft-serve/pkg/utils"
)

var _ store.OrgStore = (*orgStore)(nil)

type orgStore struct{ *handleStore }

// UpdateOrgContactEmail implements store.OrgStore.
func (*orgStore) UpdateOrgContactEmail(ctx context.Context, h db.Handler, org int64, email string) error {
	if err := utils.ValidateEmail(email); err != nil {
		return err
	}

	query := h.Rebind(`
		UPDATE organizations
		SET
		  contact_email = ?
		WHERE
		  id = ?
	`)

	_, err := h.ExecContext(ctx, query, email, org)
	return err
}

// ListOrgs implements store.OrgStore.
func (*orgStore) ListOrgs(ctx context.Context, h db.Handler, uid int64) ([]models.Organization, error) {
	var m []models.Organization
	query := h.Rebind(`
		SELECT
		  o.*,
		  h AS handle
		FROM
		  organizations o
		  JOIN handles h ON h.id = o.handle_id
		  JOIN organization_members om ON om.org_id = o.id
		WHERE
		  o.user_id = ?
	`)
	err := h.SelectContext(ctx, &m, query, uid)
	return m, err
}

// Delete implements store.OrgStore.
func (s *orgStore) DeleteOrgByID(ctx context.Context, h db.Handler, user, id int64) error {
	_, err := s.getOrgByIDWithAccess(ctx, h, user, id, access.AdminAccess)
	if err != nil {
		return err
	}
	query := h.Rebind(`DELETE FROM organizations WHERE id = ?;`)
	_, err = h.ExecContext(ctx, query, id)
	return err
}

// Create implements store.OrgStore.
func (s *orgStore) CreateOrg(ctx context.Context, h db.Handler, user int64, name, email string) (models.Organization, error) {
	if err := utils.ValidateEmail(email); err != nil {
		return models.Organization{}, err
	}

	handle, err := s.CreateHandle(ctx, h, name)
	if err != nil {
		return models.Organization{}, err
	}

	query := h.Rebind(`
		INSERT INTO
		  organizations (handle_id, contact_email, updated_at)
		VALUES
		  (?, ?, CURRENT_TIMESTAMP) RETURNING id;
	`)

	var id int64
	if err := h.GetContext(ctx, &id, query, handle, email); err != nil {
		return models.Organization{}, err
	}
	if err := s.AddUserToOrg(ctx, h, id, user, access.AdminAccess); err != nil {
		return models.Organization{}, err
	}

	return s.GetOrgByID(ctx, h, user, id)
}

func (*orgStore) UpdateUserAccessInOrg(ctx context.Context, h db.Handler, org, user int64, lvl access.AccessLevel) error {
	query := h.Rebind(`
		UPDATE organization_members
		WHERE
		  organization_id = ?
		  AND user_id = ?
		SET
		  access_level = ?
	`)
	_, err := h.ExecContext(ctx, query, org, user, lvl)
	return err
}

func (*orgStore) RemoveUserFromOrg(ctx context.Context, h db.Handler, org, user int64) error {
	query := h.Rebind(`
		DELETE FROM organization_members
		WHERE
		  organization_id = ?
		  AND user_id = ?
	`)
	_, err := h.ExecContext(ctx, query, org, user)
	return err
}

func (*orgStore) AddUserToOrg(ctx context.Context, h db.Handler, org, user int64, lvl access.AccessLevel) error {
	query := h.Rebind(`
		INSERT INTO
		  organization_members (
		    organization_id,
		    user_id,
		    access_level,
		    updated_at
		  )
		VALUES
		  (?, ?, ?, CURRENT_TIMESTAMP);
	`)
	_, err := h.ExecContext(ctx, query, org, user, lvl)
	return err
}

// FindByName implements store.OrgStore.
func (*orgStore) FindOrgByHandle(ctx context.Context, h db.Handler, user int64, name string) (models.Organization, error) {
	var m models.Organization
	query := h.Rebind(`
		SELECT
		  o.*,
		  h AS handle
		FROM
		  organizations o
		  JOIN handles h ON h.id = o.handle_id
		  JOIN organization_members om ON om.organization_id = o.id
		WHERE
		  om.user_id = ?
		  AND h.handle = ?;
	`)
	err := h.GetContext(ctx, &m, query, user, name)
	return m, err
}

// GetByID implements store.OrgStore.
func (s *orgStore) GetOrgByID(ctx context.Context, h db.Handler, user, id int64) (models.Organization, error) {
	return s.getOrgByIDWithAccess(ctx, h, user, id, access.ReadOnlyAccess)
}

func (*orgStore) getOrgByIDWithAccess(ctx context.Context, h db.Handler, user, id int64, level access.AccessLevel) (models.Organization, error) {
	var m models.Organization
	query := h.Rebind(`
		SELECT
		  o.*,
		  h AS handle
		FROM
		  organizations o
		  JOIN handles h ON h.id = o.handle_id
		  JOIN organization_members om ON om.organization_id = o.id
		WHERE
		  om.user_id = ?
		  AND id = ?
		  AND om.access_level >= ?;
	`)
	err := h.GetContext(ctx, &m, query, user, id, level)
	return m, err
}
