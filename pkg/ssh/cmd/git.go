package cmd

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"charm.land/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/git"
	"github.com/charmbracelet/soft-serve/pkg/lfs"
	"github.com/charmbracelet/soft-serve/pkg/proto"
	"github.com/charmbracelet/soft-serve/pkg/sshutils"
	"github.com/charmbracelet/soft-serve/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/cobra"
)

var (
	uploadPackCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "upload_pack_total",
		Help:      "The total number of git-upload-pack requests",
	}, []string{"repo"})

	receivePackCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "receive_pack_total",
		Help:      "The total number of git-receive-pack requests",
	}, []string{"repo"})

	uploadArchiveCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "upload_archive_total",
		Help:      "The total number of git-upload-archive requests",
	}, []string{"repo"})

	lfsAuthenticateCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "lfs_authenticate_total",
		Help:      "The total number of git-lfs-authenticate requests",
	}, []string{"repo", "operation"})

	lfsTransferCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "lfs_transfer_total",
		Help:      "The total number of git-lfs-transfer requests",
	}, []string{"repo", "operation"})

	uploadPackSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "upload_pack_seconds_total",
		Help:      "The total time spent on git-upload-pack requests",
	}, []string{"repo"})

	receivePackSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "receive_pack_seconds_total",
		Help:      "The total time spent on git-receive-pack requests",
	}, []string{"repo"})

	uploadArchiveSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "upload_archive_seconds_total",
		Help:      "The total time spent on git-upload-archive requests",
	}, []string{"repo"})

	lfsAuthenticateSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "lfs_authenticate_seconds_total",
		Help:      "The total time spent on git-lfs-authenticate requests",
	}, []string{"repo", "operation"})

	lfsTransferSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "lfs_transfer_seconds_total",
		Help:      "The total time spent on git-lfs-transfer requests",
	}, []string{"repo", "operation"})

	createRepoCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "ssh",
		Name:      "create_repo_total",
		Help:      "The total number of create repo requests",
	}, []string{"repo"})
)

// GitUploadPackCommand returns a cobra command for git-upload-pack.
func GitUploadPackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-upload-pack REPO",
		Short:  "Git upload pack",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE:   gitRunE,
	}

	return cmd
}

// GitUploadArchiveCommand returns a cobra command for git-upload-archive.
func GitUploadArchiveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-upload-archive REPO",
		Short:  "Git upload archive",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE:   gitRunE,
	}

	return cmd
}

// GitReceivePackCommand returns a cobra command for git-receive-pack.
func GitReceivePackCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-receive-pack REPO",
		Short:  "Git receive pack",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		RunE:   gitRunE,
	}

	return cmd
}

// GitLFSAuthenticateCommand returns a cobra command for git-lfs-authenticate.
func GitLFSAuthenticateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-lfs-authenticate REPO OPERATION",
		Short:  "Git LFS authenticate",
		Args:   cobra.ExactArgs(2),
		Hidden: true,
		RunE:   gitRunE,
	}

	return cmd
}

// GitLFSTransfer returns a cobra command for git-lfs-transfer.
func GitLFSTransfer() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "git-lfs-transfer REPO OPERATION",
		Short:  "Git LFS transfer",
		Args:   cobra.ExactArgs(2),
		Hidden: true,
		RunE:   gitRunE,
	}

	return cmd
}

// gitDenyRead is denyReadAs for git services: ErrNotAuthed when anonymous,
// ErrInvalidRepo when registered.
func gitDenyRead(user proto.User) error {
	return denyReadAs(user == nil, git.ErrNotAuthed, git.ErrInvalidRepo)
}

// gitAccess authorizes a git service that needs level need on repo, which
// is nil when the caller may not read it (see backend.Readable).
func gitAccess(user proto.User, repo proto.Repository, have, need access.AccessLevel) error {
	if repo == nil || have < access.ReadOnlyAccess {
		return gitDenyRead(user)
	}
	if have < need {
		return git.ErrNotAuthed
	}
	return nil
}

func gitRunE(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cfg := config.FromContext(ctx)
	be := backend.FromContext(ctx)
	logger := log.FromContext(ctx)
	start := time.Now()

	// repo should be in the form of "repo.git"
	name := utils.SanitizeRepo(args[0])
	pk := sshutils.PublicKeyFromContext(ctx)
	ak := sshutils.MarshalAuthorizedKey(pk)
	user := proto.UserFromContext(ctx)
	// git bare repositories should end in ".git"
	// https://git-scm.com/docs/gitrepository-layout
	repoDir := name + ".git"
	reposDir := filepath.Join(cfg.DataPath, "repos")
	if err := git.EnsureWithin(reposDir, repoDir); err != nil {
		return err
	}

	// Set repo in context
	repo, accessLevel := be.Readable(ctx, name, user)
	ctx = proto.WithRepositoryContext(ctx, repo)

	// Environment variables to pass down to git hooks.
	envs := []string{
		"SOFT_SERVE_REPO_NAME=" + name,
		"SOFT_SERVE_REPO_PATH=" + filepath.Join(reposDir, repoDir),
		"SOFT_SERVE_PUBLIC_KEY=" + ak,
		"SOFT_SERVE_LOG_PATH=" + filepath.Join(cfg.DataPath, "log", "hooks.log"),
	}

	if user != nil {
		envs = append(envs,
			"SOFT_SERVE_USERNAME="+user.Username(),
		)
	}

	envs = append(envs, cfg.Environ()...)

	// Add GIT_PROTOCOL from session.
	if sess := sshutils.SessionFromContext(ctx); sess != nil {
		for _, env := range sess.Environ() {
			if strings.HasPrefix(env, "GIT_PROTOCOL=") {
				envs = append(envs, env)
				break
			}
		}
	}

	repoPath := filepath.Join(reposDir, repoDir)
	service := git.Service(cmd.Name())
	stdin := cmd.InOrStdin()
	stdout := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	scmd := git.ServiceCommand{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
		Env:    envs,
		Dir:    repoPath,
	}

	switch service {
	case git.ReceivePackService:
		receivePackCounter.WithLabelValues(name).Inc()
		defer func() {
			receivePackSeconds.WithLabelValues(name).Add(time.Since(start).Seconds())
		}()
		// A missing name is created for a caller who may write; anything
		// else goes through the same check as every other service.
		created := repo == nil && accessLevel >= access.ReadWriteAccess
		if created {
			if _, err := be.CreateRepository(ctx, name, user, proto.RepositoryOptions{Private: false}); err != nil {
				log.Errorf("failed to create repo: %s", err)
				return err
			}
			createRepoCounter.WithLabelValues(name).Inc()
		} else if err := gitAccess(user, repo, accessLevel, access.ReadWriteAccess); err != nil {
			return err
		}

		if err := service.Handler(ctx, scmd); err != nil {
			logger.Error("failed to handle git service", "service", service, "err", err, "repo", name)
			if created {
				// If the repo was created, but the request failed, delete it.
				be.DeleteRepository(ctx, name) //nolint: errcheck
			}

			return git.ErrSystemMalfunction
		}

		if err := git.EnsureDefaultBranch(ctx, scmd.Dir); err != nil {
			logger.Error("failed to ensure default branch", "err", err, "repo", name)
			return git.ErrSystemMalfunction
		}

		receivePackCounter.WithLabelValues(name).Inc()

		return nil
	case git.UploadPackService, git.UploadArchiveService:
		if err := gitAccess(user, repo, accessLevel, access.ReadOnlyAccess); err != nil {
			return err
		}

		switch service {
		case git.UploadArchiveService:
			uploadArchiveCounter.WithLabelValues(name).Inc()
			defer func() {
				uploadArchiveSeconds.WithLabelValues(name).Add(time.Since(start).Seconds())
			}()
		default:
			uploadPackCounter.WithLabelValues(name).Inc()
			defer func() {
				uploadPackSeconds.WithLabelValues(name).Add(time.Since(start).Seconds())
			}()
		}

		if err := service.Handler(ctx, scmd); err != nil {
			logger.Error("failed to handle git service", "service", service, "err", err, "repo", name)
			return git.ErrSystemMalfunction
		}

		return nil
	case git.LFSTransferService, git.LFSAuthenticateService:
		operation := args[1]
		var need access.AccessLevel
		switch operation {
		case lfs.OperationDownload:
			need = access.ReadOnlyAccess
		case lfs.OperationUpload:
			need = access.ReadWriteAccess
		default:
			return git.ErrInvalidRequest
		}

		if err := gitAccess(user, repo, accessLevel, need); err != nil {
			return err
		}

		scmd.Args = []string{
			name,
			args[1],
		}

		switch service {
		case git.LFSTransferService:
			lfsTransferCounter.WithLabelValues(name, operation).Inc()
			defer func() {
				lfsTransferSeconds.WithLabelValues(name, operation).Add(time.Since(start).Seconds())
			}()
		default:
			lfsAuthenticateCounter.WithLabelValues(name, operation).Inc()
			defer func() {
				lfsAuthenticateSeconds.WithLabelValues(name, operation).Add(time.Since(start).Seconds())
			}()
		}

		if err := service.Handler(ctx, scmd); err != nil {
			logger.Error("failed to handle lfs service", "service", service, "err", err, "repo", name)
			return git.ErrSystemMalfunction
		}

		return nil
	}

	return errors.New("unsupported git service")
}
