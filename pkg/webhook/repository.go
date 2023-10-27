package webhook

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

// RepositoryEvent is a repository payload.
type RepositoryEvent struct {
	Common

	// Action is the repository event action.
	Action RepositoryEventAction `json:"action" url:"action"`
}

// RepositoryEventAction is a repository event action.
type RepositoryEventAction string

const (
	// RepositoryEventActionDelete is a repository deleted event.
	RepositoryEventActionDelete RepositoryEventAction = "delete"
	// RepositoryEventActionRename is a repository renamed event.
	RepositoryEventActionRename RepositoryEventAction = "rename"
	// RepositoryEventActionVisibilityChange is a repository visibility changed event.
	RepositoryEventActionVisibilityChange RepositoryEventAction = "visibility_change"
	// RepositoryEventActionDefaultBranchChange is a repository default branch changed event.
	RepositoryEventActionDefaultBranchChange RepositoryEventAction = "default_branch_change"
)

// NewRepositoryEvent sends a repository event.
func NewRepositoryEvent(ctx context.Context, user proto.User, repo proto.Repository, action RepositoryEventAction) (RepositoryEvent, error) {
	var event Event
	switch action {
	case RepositoryEventActionVisibilityChange:
		event = EventRepositoryVisibilityChange
	default:
		event = EventRepository
	}

	payload := RepositoryEvent{
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

	cfg := config.FromContext(ctx)
	payload.Repository.HTTPURL = repoURL(cfg.HTTP.PublicURL, repo.Name())
	payload.Repository.SSHURL = repoURL(cfg.SSH.PublicURL, repo.Name())
	payload.Repository.GitURL = repoURL(cfg.Git.PublicURL, repo.Name())

	// Find repo owner.
	dbx := db.FromContext(ctx)
	datastore := store.FromContext(ctx)
	owner, err := datastore.GetUserByID(ctx, dbx, repo.UserID())
	if err != nil {
		return RepositoryEvent{}, db.WrapError(err)
	}

	payload.Repository.Owner.ID = owner.ID
	payload.Repository.Owner.Username = owner.Username
	payload.Repository.DefaultBranch, _ = proto.RepositoryDefaultBranch(repo)

	return payload, nil
}
