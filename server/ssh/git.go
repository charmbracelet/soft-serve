package ssh

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/access"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/charmbracelet/soft-serve/server/lfs"
	"github.com/charmbracelet/soft-serve/server/proto"
	"github.com/charmbracelet/soft-serve/server/sshutils"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/charmbracelet/ssh"
)

func handleGit(s ssh.Session) {
	ctx := s.Context()
	cfg := config.FromContext(ctx)
	be := backend.FromContext(ctx)
	logger := log.FromContext(ctx)
	cmdLine := s.Command()
	start := time.Now()

	// repo should be in the form of "repo.git"
	name := utils.SanitizeRepo(cmdLine[1])
	pk := s.PublicKey()
	ak := sshutils.MarshalAuthorizedKey(pk)
	user := proto.UserFromContext(ctx)
	accessLevel := be.AccessLevelForUser(ctx, name, user)
	// git bare repositories should end in ".git"
	// https://git-scm.com/docs/gitrepository-layout
	repoDir := name + ".git"
	reposDir := filepath.Join(cfg.DataPath, "repos")
	if err := git.EnsureWithin(reposDir, repoDir); err != nil {
		sshFatal(s, err)
		return
	}

	// Set repo in context
	repo, _ := be.Repository(ctx, name)
	ctx.SetValue(proto.ContextKeyRepository, repo)

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

	// Add ssh session & config environ
	envs = append(envs, s.Environ()...)
	envs = append(envs, cfg.Environ()...)

	repoPath := filepath.Join(reposDir, repoDir)
	service := git.Service(cmdLine[0])
	cmd := git.ServiceCommand{
		Stdin:  s,
		Stdout: s,
		Stderr: s.Stderr(),
		Env:    envs,
		Dir:    repoPath,
	}

	logger.Debug("git middleware", "cmd", service, "access", accessLevel.String())

	switch service {
	case git.ReceivePackService:
		receivePackCounter.WithLabelValues(name).Inc()
		defer func() {
			receivePackSeconds.WithLabelValues(name).Add(time.Since(start).Seconds())
		}()
		if accessLevel < access.ReadWriteAccess {
			sshFatal(s, git.ErrNotAuthed)
			return
		}
		if repo == nil {
			if _, err := be.CreateRepository(ctx, name, proto.RepositoryOptions{Private: false}); err != nil {
				log.Errorf("failed to create repo: %s", err)
				sshFatal(s, err)
				return
			}
			createRepoCounter.WithLabelValues(name).Inc()
		}

		if err := service.Handler(ctx, cmd); err != nil {
			sshFatal(s, git.ErrSystemMalfunction)
		}

		if err := git.EnsureDefaultBranch(ctx, cmd); err != nil {
			sshFatal(s, git.ErrSystemMalfunction)
		}

		receivePackCounter.WithLabelValues(name).Inc()
		return
	case git.UploadPackService, git.UploadArchiveService:
		if accessLevel < access.ReadOnlyAccess {
			sshFatal(s, git.ErrNotAuthed)
			return
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

		err := service.Handler(ctx, cmd)
		if errors.Is(err, git.ErrInvalidRepo) {
			sshFatal(s, git.ErrInvalidRepo)
		} else if err != nil {
			logger.Error("git middleware", "err", err)
			sshFatal(s, git.ErrSystemMalfunction)
		}

		return
	case git.LFSTransferService, git.LFSAuthenticateService:
		if !cfg.LFS.Enabled {
			return
		}

		if service == git.LFSTransferService && !cfg.LFS.SSHEnabled {
			return
		}

		if accessLevel < access.ReadWriteAccess {
			sshFatal(s, git.ErrNotAuthed)
			return
		}

		if len(cmdLine) != 3 ||
			(cmdLine[2] != lfs.OperationDownload && cmdLine[2] != lfs.OperationUpload) {
			sshFatal(s, git.ErrInvalidRequest)
			return
		}

		cmd.Args = []string{
			name,
			cmdLine[2],
		}

		if err := service.Handler(ctx, cmd); err != nil {
			logger.Error("git middleware", "err", err)
			sshFatal(s, git.ErrSystemMalfunction)
			return
		}
	}
}
