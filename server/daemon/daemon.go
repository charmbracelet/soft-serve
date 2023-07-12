package daemon

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/charmbracelet/soft-serve/server/store"
	"github.com/charmbracelet/soft-serve/server/utils"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	uploadPackGitCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "git_upload_pack_total",
		Help:      "The total number of git-upload-pack requests",
	}, []string{"repo"})

	uploadArchiveGitCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "soft_serve",
		Subsystem: "git",
		Name:      "git_upload_archive_total",
		Help:      "The total number of git-upload-archive requests",
	}, []string{"repo"})
)

var (

	// ErrServerClosed indicates that the server has been closed.
	ErrServerClosed = fmt.Errorf("git: %w", net.ErrClosed)
)

// GitDaemon represents a Git daemon.
type GitDaemon struct {
	ctx      context.Context
	listener net.Listener
	addr     string
	finished chan struct{}
	conns    connections
	cfg      *config.Config
	be       *backend.Backend
	wg       sync.WaitGroup
	once     sync.Once
	logger   *log.Logger
}

// NewDaemon returns a new Git daemon.
func NewGitDaemon(ctx context.Context) (*GitDaemon, error) {
	cfg := config.FromContext(ctx)
	addr := cfg.Git.ListenAddr
	d := &GitDaemon{
		ctx:      ctx,
		addr:     addr,
		finished: make(chan struct{}, 1),
		cfg:      cfg,
		be:       backend.FromContext(ctx),
		conns:    connections{m: make(map[net.Conn]struct{})},
		logger:   log.FromContext(ctx).WithPrefix("gitdaemon"),
	}
	listener, err := net.Listen("tcp", d.addr)
	if err != nil {
		return nil, err
	}
	d.listener = listener
	return d, nil
}

// Start starts the Git TCP daemon.
func (d *GitDaemon) Start() error {
	defer d.listener.Close() // nolint: errcheck

	d.wg.Add(1)
	defer d.wg.Done()

	var tempDelay time.Duration
	for {
		conn, err := d.listener.Accept()
		if err != nil {
			select {
			case <-d.finished:
				return ErrServerClosed
			default:
				d.logger.Debugf("git: error accepting connection: %v", err)
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			return err
		}

		// Close connection if there are too many open connections.
		if d.conns.Size()+1 >= d.cfg.Git.MaxConnections {
			d.logger.Debugf("git: max connections reached, closing %s", conn.RemoteAddr())
			d.fatal(conn, git.ErrMaxConnections)
			continue
		}

		d.wg.Add(1)
		go func() {
			d.handleClient(conn)
			d.wg.Done()
		}()
	}
}

func (d *GitDaemon) fatal(c net.Conn, err error) {
	git.WritePktline(c, err)
	if err := c.Close(); err != nil {
		d.logger.Debugf("git: error closing connection: %v", err)
	}
}

// handleClient handles a git protocol client.
func (d *GitDaemon) handleClient(conn net.Conn) {
	ctx, cancel := context.WithCancel(context.Background())
	idleTimeout := time.Duration(d.cfg.Git.IdleTimeout) * time.Second
	c := &serverConn{
		Conn:          conn,
		idleTimeout:   idleTimeout,
		closeCanceler: cancel,
	}
	if d.cfg.Git.MaxTimeout > 0 {
		dur := time.Duration(d.cfg.Git.MaxTimeout) * time.Second
		c.maxDeadline = time.Now().Add(dur)
	}
	d.conns.Add(c)
	defer func() {
		d.conns.Close(c)
	}()

	readc := make(chan struct{}, 1)
	s := pktline.NewScanner(c)
	go func() {
		if !s.Scan() {
			if err := s.Err(); err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					d.fatal(c, git.ErrTimeout)
				} else {
					d.logger.Debugf("git: error scanning pktline: %v", err)
					d.fatal(c, git.ErrSystemMalfunction)
				}
			}
			return
		}
		readc <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			d.logger.Debugf("git: connection context error: %v", err)
		}
		return
	case <-readc:
		line := s.Bytes()
		split := bytes.SplitN(line, []byte{' '}, 2)
		if len(split) != 2 {
			d.fatal(c, git.ErrInvalidRequest)
			return
		}

		var handler git.ServiceHandler
		var counter *prometheus.CounterVec
		service := git.Service(split[0])
		switch service {
		case git.UploadPackService:
			handler = git.UploadPack
			counter = uploadPackGitCounter
		case git.UploadArchiveService:
			handler = git.UploadArchive
			counter = uploadArchiveGitCounter
		default:
			d.fatal(c, git.ErrInvalidRequest)
			return
		}

		opts := bytes.SplitN(split[1], []byte{0}, 3)
		if len(opts) < 2 {
			d.fatal(c, git.ErrInvalidRequest) // nolint: errcheck
			return
		}

		host := strings.TrimPrefix(string(opts[1]), "host=")
		extraParams := map[string]string{}

		if len(opts) > 2 {
			buf := bytes.TrimPrefix(opts[2], []byte{0})
			for _, o := range bytes.Split(buf, []byte{0}) {
				opt := string(o)
				if opt == "" {
					continue
				}

				kv := strings.SplitN(opt, "=", 2)
				if len(kv) != 2 {
					d.logger.Errorf("git: invalid option %q", opt)
					continue
				}

				extraParams[kv[0]] = kv[1]
			}

			version := extraParams["version"]
			if version != "" {
				d.logger.Debugf("git: protocol version %s", version)
			}
		}

		be := d.be
		if !be.AllowKeyless(ctx) {
			d.fatal(c, git.ErrNotAuthed)
			return
		}

		name := utils.SanitizeRepo(string(opts[0]))
		d.logger.Debugf("git: connect %s %s %s", c.RemoteAddr(), service, name)
		defer d.logger.Debugf("git: disconnect %s %s %s", c.RemoteAddr(), service, name)

		// git bare repositories should end in ".git"
		// https://git-scm.com/docs/gitrepository-layout
		repo := name + ".git"
		reposDir := filepath.Join(d.cfg.DataPath, "repos")
		if err := git.EnsureWithin(reposDir, repo); err != nil {
			d.logger.Debugf("git: error ensuring repo path: %v", err)
			d.fatal(c, git.ErrInvalidRepo)
			return
		}

		if _, err := d.be.Repository(ctx, repo); err != nil {
			d.fatal(c, git.ErrInvalidRepo)
			return
		}

		auth := be.AccessLevel(ctx, name, "")
		if auth < store.ReadOnlyAccess {
			d.fatal(c, git.ErrNotAuthed)
			return
		}

		// Environment variables to pass down to git hooks.
		envs := []string{
			"SOFT_SERVE_REPO_NAME=" + name,
			"SOFT_SERVE_REPO_PATH=" + filepath.Join(reposDir, repo),
			"SOFT_SERVE_HOST=" + host,
		}

		// Add git protocol environment variable.
		if len(extraParams) > 0 {
			var gitProto string
			for k, v := range extraParams {
				if len(gitProto) > 0 {
					gitProto += ":"
				}
				gitProto += k + "=" + v
			}
			envs = append(envs, "GIT_PROTOCOL="+gitProto)
		}

		envs = append(envs, d.cfg.Environ()...)

		cmd := git.ServiceCommand{
			Stdin:  c,
			Stdout: c,
			Stderr: c,
			Env:    envs,
			Dir:    filepath.Join(reposDir, repo),
		}

		if err := handler(ctx, cmd); err != nil {
			d.logger.Debugf("git: error handling request: %v", err)
			d.fatal(c, err)
			return
		}

		counter.WithLabelValues(name)
	}
}

// Close closes the underlying listener.
func (d *GitDaemon) Close() error {
	d.once.Do(func() { close(d.finished) })
	err := d.listener.Close()
	d.conns.CloseAll()
	return err
}

// Shutdown gracefully shuts down the daemon.
func (d *GitDaemon) Shutdown(ctx context.Context) error {
	d.once.Do(func() { close(d.finished) })
	err := d.listener.Close()
	finished := make(chan struct{}, 1)
	go func() {
		d.wg.Wait()
		finished <- struct{}{}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-finished:
		return err
	}
}
