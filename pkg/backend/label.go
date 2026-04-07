package backend

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/store"
)

// namedColors maps human-friendly color names to their hex equivalents.
var namedColors = map[string]string{
	"red":     "#ff0000",
	"blue":    "#0000ff",
	"yellow":  "#ffff00",
	"green":   "#008000",
	"magenta": "#ff00ff",
	"pink":    "#ffc0cb",
	"white":   "#ffffff",
	"black":   "#000000",
}

// normalizeColor resolves a named color (e.g. "red") to its hex value.
// Values that are not a known name are returned unchanged, so bare hex
// strings like "ff0000" or "#ff0000" pass through as-is.
func normalizeColor(c string) string {
	if hex, ok := namedColors[strings.ToLower(strings.TrimSpace(c))]; ok {
		return hex
	}
	return c
}

// CreateLabel creates a new label for a repository.
// Name must be non-empty and must not contain spaces.
func (b *Backend) CreateLabel(ctx context.Context, repoName, name, color, description string) (proto.Label, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("label name cannot be empty")
	}
	if strings.Contains(name, " ") {
		return nil, fmt.Errorf("label name cannot contain spaces")
	}

	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	var id int64
	if err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		id, err = store.FromContext(ctx).CreateLabel(ctx, tx, repo.ID(), name, normalizeColor(color), description)
		return err
	}); err != nil {
		if errors.Is(err, db.ErrDuplicateKey) {
			return nil, fmt.Errorf("label %q already exists in repository %s", name, repoName)
		}
		return nil, err
	}

	return b.getLabelByID(ctx, id)
}

// GetLabel retrieves a label by name from a repository.
func (b *Backend) GetLabel(ctx context.Context, repoName, name string) (proto.Label, error) {
	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	m, err := store.FromContext(ctx).GetLabelByName(ctx, b.db, repo.ID(), name)
	if err != nil {
		return nil, fmt.Errorf("label %q not found in repository %s", name, repoName)
	}

	return proto.NewLabel(m), nil
}

// ListLabels returns all labels for a repository.
func (b *Backend) ListLabels(ctx context.Context, repoName string) ([]proto.Label, error) {
	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return nil, err
	}

	ms, err := store.FromContext(ctx).GetLabelsByRepoID(ctx, b.db, repo.ID())
	if err != nil {
		return nil, err
	}

	labels := make([]proto.Label, len(ms))
	for i, m := range ms {
		labels[i] = proto.NewLabel(m)
	}
	return labels, nil
}

// UpdateLabel updates a label's fields. Only flags that are Changed should be
// passed with new values; unchanged fields should carry the current values.
func (b *Backend) UpdateLabel(ctx context.Context, repoName string, labelID int64, name, color, description string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("label name cannot be empty")
	}
	if strings.Contains(name, " ") {
		return fmt.Errorf("label name cannot contain spaces")
	}

	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		err := store.FromContext(ctx).UpdateLabel(ctx, tx, labelID, repo.ID(), name, normalizeColor(color), description)
		if errors.Is(err, db.ErrDuplicateKey) {
			return fmt.Errorf("label %q already exists in repository %s", name, repoName)
		}
		return err
	})
}

// DeleteLabel deletes a label from a repository.
func (b *Backend) DeleteLabel(ctx context.Context, repoName string, labelID int64) error {
	repo, err := b.Repository(ctx, repoName)
	if err != nil {
		return err
	}

	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return store.FromContext(ctx).DeleteLabel(ctx, tx, labelID, repo.ID())
	})
}

// GetIssueLabels returns all labels attached to an issue.
func (b *Backend) GetIssueLabels(ctx context.Context, issueID int64) ([]proto.Label, error) {
	ms, err := store.FromContext(ctx).GetLabelsByIssueID(ctx, b.db, issueID)
	if err != nil {
		return nil, err
	}

	labels := make([]proto.Label, len(ms))
	for i, m := range ms {
		labels[i] = proto.NewLabel(m)
	}
	return labels, nil
}

// AddLabelToIssue attaches a label to an issue. If the label is already
// attached, this is a no-op.
func (b *Backend) AddLabelToIssue(ctx context.Context, issueID, labelID int64) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		err := store.FromContext(ctx).AddLabelToIssue(ctx, tx, issueID, labelID)
		if errors.Is(err, db.ErrDuplicateKey) {
			return nil // already attached — idempotent
		}
		return err
	})
}

// RemoveLabelFromIssue detaches a label from an issue.
func (b *Backend) RemoveLabelFromIssue(ctx context.Context, issueID, labelID int64) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return store.FromContext(ctx).RemoveLabelFromIssue(ctx, tx, issueID, labelID)
	})
}

// getLabelByID is a helper for looking up a label by its database ID.
func (b *Backend) getLabelByID(ctx context.Context, id int64) (proto.Label, error) {
	m, err := store.FromContext(ctx).GetLabelByID(ctx, b.db, id)
	if err != nil {
		return nil, err
	}
	return proto.NewLabel(m), nil
}
