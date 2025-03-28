package daemon

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log/v2"
	"github.com/charmbracelet/soft-serve/pkg/access"
	"github.com/charmbracelet/soft-serve/pkg/backend"
	"github.com/charmbracelet/soft-serve/pkg/config"
	"github.com/charmbracelet/soft-serve/pkg/git"
	"github.com/charmbracelet/soft-serve/pkg/utils"
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

// ErrServerClosed indicates that the server has been closed.
var ErrServerClosed = fmt.Errorf("git: %w", net.ErrClosed)

// GitDaemon represents a Git daemon.
type GitDaemon struct {
	ctx      context.Context
	addr     string
	finished chan struct{}
	conns    connections
	cfg      *config.Config
	be       *backend.Backend
	wg       sync.WaitGroup
	once     sync.Once
	logger   *log.Logger
	done     atomic.Bool // indicates if the server has been closed
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
	return d, nil
}

// ListenAndServe starts the Git TCP daemon.
func (d *GitDaemon) ListenAndServe() error {
	if d.done.Load() {
		return ErrServerClosed
	}
	listener, err := net.Listen("tcp", d.addr)
	if err != nil {
		return err
	}
	return d.Serve(listener)
}

// Serve listens on the TCP network address and serves Git requests.
func (d *GitDaemon) Serve(listener net.Listener) error {
	if d.done.Load() {
		return ErrServerClosed
	}

	d.wg.Add(1)
	defer d.wg.Done()
	defer listener.Close() //nolint:errcheck

	var tempDelay time.Duration
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-d.finished:
				return ErrServerClosed
			default:
				d.logger.Debugf("git: error accepting connection: %v", err)
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() { // nolint: staticcheck
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max { //nolint:revive
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
	git.WritePktlineErr(c, err) // nolint: errcheck
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
		d.conns.Close(c) // nolint: errcheck
	}()

	errc := make(chan error, 1)

	s := pktline.NewScanner(c)
	go func() {
		if !s.Scan() {
			if err := s.Err(); err != nil {
				errc <- err
			}
		}
		errc <- nil
	}()

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			d.logger.Debugf("git: connection context error: %v", err)
			d.fatal(c, git.ErrTimeout)
		}
		return
	case err := <-errc:
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			d.fatal(c, git.ErrTimeout)
			return
		} else if err != nil {
			d.logger.Debugf("git: error scanning pktline: %v", err)
			d.fatal(c, git.ErrSystemMalfunction)
			return
		}

		line := s.Bytes()
		split := bytes.SplitN(line, []byte{' '}, 2)
		if len(split) != 2 {
			d.fatal(c, git.ErrInvalidRequest)
			return
		}

		var counter *prometheus.CounterVec
		service := git.Service(split[0])
		switch service {
		case git.UploadPackService:
			counter = uploadPackGitCounter
		case git.UploadArchiveService:
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
		if auth < access.ReadOnlyAccess {
			d.fatal(c, git.ErrNotAuthed)
			return
		}

		// Environment variables to pass down to git hooks.
		envs := []string{
			"SOFT_SERVE_REPO_NAME=" + name,
			"SOFT_SERVE_REPO_PATH=" + filepath.Join(reposDir, repo),
			"SOFT_SERVE_HOST=" + host,
			"SOFT_SERVE_LOG_PATH=" + filepath.Join(d.cfg.DataPath, "log", "hooks.log"),
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

		if err := service.Handler(ctx, cmd); err != nil {
			d.logger.Debugf("git: error handling request: %v", err)
			d.fatal(c, err)
			return
		}

		counter.WithLabelValues(name)
	}
}

// Close closes the underlying listener.
func (d *GitDaemon) Close() error {
	err := d.closeListener()
	d.conns.CloseAll() // nolint: errcheck
	return err
}

// closeListener closes the listener and the finished channel.
func (d *GitDaemon) closeListener() error {
	if d.done.Load() {
		return ErrServerClosed
	}
	d.once.Do(func() {
		close(d.finished)
		d.done.Store(true)
	})
	return nil
}

// Shutdown gracefully shuts down the daemon.
func (d *GitDaemon) Shutdown(ctx context.Context) error {
	if d.done.Load() {
		return ErrServerClosed
	}

	err := d.closeListener()
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
