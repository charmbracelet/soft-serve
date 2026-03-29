package backend

import (
	"context"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/soft-serve/pkg/db"
	"github.com/charmbracelet/soft-serve/pkg/db/models"
	"github.com/charmbracelet/soft-serve/pkg/proto"
)

// AddPushMirror adds a push mirror to a repository.
func (b *Backend) AddPushMirror(ctx context.Context, repo proto.Repository, name, remoteURL string) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.CreatePushMirror(ctx, tx, repo.ID(), name, remoteURL)
	})
}

// RemovePushMirror removes a push mirror from a repository.
func (b *Backend) RemovePushMirror(ctx context.Context, repo proto.Repository, name string) error {
	return b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		return b.store.DeletePushMirror(ctx, tx, repo.ID(), name)
	})
}

// ListPushMirrors lists all push mirrors for a repository.
func (b *Backend) ListPushMirrors(ctx context.Context, repo proto.Repository) ([]models.PushMirror, error) {
	var mirrors []models.PushMirror
	err := b.db.TransactionContext(ctx, func(tx *db.Tx) error {
		var err error
		mirrors, err = b.store.GetPushMirrorsByRepoID(ctx, tx, repo.ID())
		return err
	})
	return mirrors, err
}

// PushMirrors triggers an async push to all enabled mirrors for a repository.
// Called from the post-receive hook. Errors are logged but not fatal.
func (b *Backend) PushMirrors(ctx context.Context, repo proto.Repository) {
	mirrors, err := b.ListPushMirrors(ctx, repo)
	if err != nil {
		b.logger.Warn("push-mirror: failed to list mirrors", "repo", repo.Name(), "err", err)
		return
	}
	repoPath := b.repoPath(repo.Name())
	const mirrorPushTimeout = 10 * time.Minute
	const maxConcurrentPushes = 5
	sem := make(chan struct{}, maxConcurrentPushes)
	for _, m := range mirrors {
		if !m.Enabled {
			continue
		}
		if u, err := url.Parse(m.RemoteURL); err == nil {
			scheme := strings.ToLower(u.Scheme)
			if scheme == "file" {
				b.logger.Warn("push mirror: blocked file:// URL", "url", m.RemoteURL)
				continue // skip this mirror
			}
		}
		sem <- struct{}{} // acquire
		go func(m models.PushMirror) {
			defer func() { <-sem }() // release
			mirrorCtx, cancel := context.WithTimeout(ctx, mirrorPushTimeout)
			defer cancel()
			cmd := exec.CommandContext(mirrorCtx, "git", "push", "--mirror", m.RemoteURL)
			cmd.Dir = repoPath
			if out, err := cmd.CombinedOutput(); err != nil {
				b.logger.Warn("push-mirror: push failed", "repo", repo.Name(), "mirror", m.Name, "err", err, "output", string(out))
			} else {
				b.logger.Info("push-mirror: pushed", "repo", repo.Name(), "mirror", m.Name)
			}
		}(m)
	}
}
