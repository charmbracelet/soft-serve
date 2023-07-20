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

	user, ok := ctx.Value(proto.ContextKeyUser).(proto.User)
	if !ok {
		sshFatal(s, errors.New("no user in context"))
		return
	}

	// repo should be in the form of "repo.git"
	name := utils.SanitizeRepo(cmdLine[1])
	pk := s.PublicKey()
	ak := sshutils.MarshalAuthorizedKey(pk)
	accessLevel := be.AccessLevelForUser(ctx, name, user)
	// git bare repositories should end in ".git"
	// https://git-scm.com/docs/gitrepository-layout
	repo := name + ".git"
	reposDir := filepath.Join(cfg.DataPath, "repos")
	if err := git.EnsureWithin(reposDir, repo); err != nil {
		sshFatal(s, err)
		return
	}

	// Environment variables to pass down to git hooks.
	envs := []string{
		"SOFT_SERVE_REPO_NAME=" + name,
		"SOFT_SERVE_REPO_PATH=" + filepath.Join(reposDir, repo),
		"SOFT_SERVE_PUBLIC_KEY=" + ak,
		"SOFT_SERVE_USERNAME=" + user.Username(),
		"SOFT_SERVE_LOG_PATH=" + filepath.Join(cfg.DataPath, "log", "hooks.log"),
	}

	// Add ssh session & config environ
	envs = append(envs, s.Environ()...)
	envs = append(envs, cfg.Environ()...)

	repoDir := filepath.Join(reposDir, repo)
	service := git.Service(cmdLine[0])
	cmd := git.ServiceCommand{
		Stdin:  s,
		Stdout: s,
		Stderr: s.Stderr(),
		Env:    envs,
		Dir:    repoDir,
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
		if _, err := be.Repository(ctx, name); err != nil {
			if _, err := be.CreateRepository(ctx, name, proto.RepositoryOptions{Private: false}); err != nil {
				log.Errorf("failed to create repo: %s", err)
				sshFatal(s, err)
				return
			}
			createRepoCounter.WithLabelValues(name).Inc()
		}

		if err := git.ReceivePack(ctx, cmd); err != nil {
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

		handler := git.UploadPack
		switch service {
		case git.UploadArchiveService:
			handler = git.UploadArchive
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

		err := handler(ctx, cmd)
		if errors.Is(err, git.ErrInvalidRepo) {
			sshFatal(s, git.ErrInvalidRepo)
		} else if err != nil {
			logger.Error("git middleware", "err", err)
			sshFatal(s, git.ErrSystemMalfunction)
		}
	case git.LFSTransferService:
		if accessLevel < access.ReadWriteAccess {
			sshFatal(s, git.ErrNotAuthed)
			return
		}

		if len(cmdLine) != 3 ||
			(cmdLine[2] != "download" && cmdLine[2] != "upload") {
			sshFatal(s, git.ErrInvalidRequest)
			return
		}

		cmd.Args = []string{
			name,
			cmdLine[2],
		}

		if err := git.LFSTransfer(ctx, cmd); err != nil {
			logger.Error("git middleware", "err", err)
			sshFatal(s, git.ErrSystemMalfunction)
			return
		}
	}
}
