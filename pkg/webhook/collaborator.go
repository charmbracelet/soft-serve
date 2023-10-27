package webhook

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

// CollaboratorEvent is a collaborator event.
type CollaboratorEvent struct {
	Common

	// Action is the collaborator event action.
	Action CollaboratorEventAction `json:"action" url:"action"`
	// AccessLevel is the collaborator access level.
	AccessLevel access.AccessLevel `json:"access_level" url:"access_level"`
	// Collaborator is the collaborator.
	Collaborator User `json:"collaborator" url:"collaborator"`
}

// CollaboratorEventAction is a collaborator event action.
type CollaboratorEventAction string

const (
	// CollaboratorEventAdded is a collaborator added event.
	CollaboratorEventAdded CollaboratorEventAction = "added"
	// CollaboratorEventRemoved is a collaborator removed event.
	CollaboratorEventRemoved CollaboratorEventAction = "removed"
)

// NewCollaboratorEvent sends a collaborator event.
func NewCollaboratorEvent(ctx context.Context, user proto.User, repo proto.Repository, collabUsername string, action CollaboratorEventAction) (CollaboratorEvent, error) {
	event := EventCollaborator

	payload := CollaboratorEvent{
		Action: action,
		Common: Common{
			EventType: event,
			Repository: Repository{
				ID:          repo.ID(),
				Name:        repo.Name(),
				Description: repo.Description(),
				ProjectName: repo.ProjectName(),
				Private:     repo.IsPrivate(),
				CreatedAt:   repo.CreatedAt(),
				UpdatedAt:   repo.UpdatedAt(),
			},
			Sender: User{
				ID:       user.ID(),
				Username: user.Username(),
			},
		},
	}

	// Find repo owner.
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	owner, err := datastore.GetUserByID(ctx, dbx, repo.UserID())
	if err != nil {
		return CollaboratorEvent{}, db.WrapError(err)
	}

	payload.Repository.Owner.ID = owner.ID
	payload.Repository.Owner.Username = owner.Username
	payload.Repository.DefaultBranch, err = proto.RepositoryDefaultBranch(repo)
	if err != nil {
		return CollaboratorEvent{}, err
	}

	collab, err := datastore.GetCollabByUsernameAndRepo(ctx, dbx, collabUsername, repo.Name())
	if err != nil {
		return CollaboratorEvent{}, err
	}

	payload.AccessLevel = collab.AccessLevel
	payload.Collaborator.ID = collab.UserID
	payload.Collaborator.Username = collabUsername

	return payload, nil
}
