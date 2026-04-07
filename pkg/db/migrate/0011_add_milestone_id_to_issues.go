package migrate

import (
	"context"

	"github.com/charmbracelet/soft-serve/pkg/db"
)

const (
	addMilestoneIDToIssuesName    = "add_milestone_id_to_issues"
	addMilestoneIDToIssuesVersion = 11
)

var addMilestoneIDToIssues = Migration{
	Name:    addMilestoneIDToIssuesName,
	Version: addMilestoneIDToIssuesVersion,
	Migrate: func(ctx context.Context, tx *db.Tx) error {
		return migrateUp(ctx, tx, addMilestoneIDToIssuesVersion, addMilestoneIDToIssuesName)
	},
	Rollback: func(ctx context.Context, tx *db.Tx) error {
		return migrateDown(ctx, tx, addMilestoneIDToIssuesVersion, addMilestoneIDToIssuesName)
	},
}
