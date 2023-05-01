package daemon

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/backend"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git"
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

// connections synchronizes access to to a net.Conn pool.
type connections struct {
	m  map[net.Conn]struct{}
	mu sync.Mutex
}

func (m *connections) Add(c net.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[c] = struct{}{}
}

func (m *connections) Close(c net.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_ = c.Close()
	delete(m.m, c)
}

func (m *connections) Size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.m)
}

func (m *connections) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for c := range m.m {
		_ = c.Close()
		delete(m.m, c)
	}
}

// GitDaemon represents a Git daemon.
type GitDaemon struct {
	ctx      context.Context
	listener net.Listener
	addr     string
	finished chan struct{}
	conns    connections
	cfg      *config.Config
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

		gitPack := git.UploadPack
		counter := uploadPackGitCounter
		cmd := string(split[0])
		switch cmd {
		case git.UploadPackBin:
			gitPack = git.UploadPack
		case git.UploadArchiveBin:
			gitPack = git.UploadArchive
			counter = uploadArchiveGitCounter
		default:
			d.fatal(c, git.ErrInvalidRequest)
			return
		}

		opts := bytes.Split(split[1], []byte{'\x00'})
		if len(opts) == 0 {
			d.fatal(c, git.ErrInvalidRequest)
			return
		}

		if !d.cfg.Backend.AllowKeyless() {
			d.fatal(c, git.ErrNotAuthed)
			return
		}

		name := utils.SanitizeRepo(string(opts[0]))
		d.logger.Debugf("git: connect %s %s %s", c.RemoteAddr(), cmd, name)
		defer d.logger.Debugf("git: disconnect %s %s %s", c.RemoteAddr(), cmd, name)
		// git bare repositories should end in ".git"
		// https://git-scm.com/docs/gitrepository-layout
		repo := name + ".git"
		reposDir := filepath.Join(d.cfg.DataPath, "repos")
		if err := git.EnsureWithin(reposDir, repo); err != nil {
			d.fatal(c, err)
			return
		}

		auth := d.cfg.Backend.AccessLevel(name, "")
		if auth < backend.ReadOnlyAccess {
			d.fatal(c, git.ErrNotAuthed)
			return
		}

		// Environment variables to pass down to git hooks.
		envs := []string{
			"SOFT_SERVE_REPO_NAME=" + name,
			"SOFT_SERVE_REPO_PATH=" + filepath.Join(reposDir, repo),
		}

		if err := gitPack(ctx, c, c, c, filepath.Join(reposDir, repo), envs...); err != nil {
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

type serverConn struct {
	net.Conn

	idleTimeout   time.Duration
	maxDeadline   time.Time
	closeCanceler context.CancelFunc
}

func (c *serverConn) Write(p []byte) (n int, err error) {
	c.updateDeadline()
	n, err = c.Conn.Write(p)
	if _, isNetErr := err.(net.Error); isNetErr && c.closeCanceler != nil {
		c.closeCanceler()
	}
	return
}

func (c *serverConn) Read(b []byte) (n int, err error) {
	c.updateDeadline()
	n, err = c.Conn.Read(b)
	if _, isNetErr := err.(net.Error); isNetErr && c.closeCanceler != nil {
		c.closeCanceler()
	}
	return
}

func (c *serverConn) Close() (err error) {
	err = c.Conn.Close()
	if c.closeCanceler != nil {
		c.closeCanceler()
	}
	return
}

func (c *serverConn) updateDeadline() {
	switch {
	case c.idleTimeout > 0:
		idleDeadline := time.Now().Add(c.idleTimeout)
		if idleDeadline.Unix() < c.maxDeadline.Unix() || c.maxDeadline.IsZero() {
			c.Conn.SetDeadline(idleDeadline)
			return
		}
		fallthrough
	default:
		c.Conn.SetDeadline(c.maxDeadline)
	}
}
