package daemon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/soft-serve/proto"
	"github.com/charmbracelet/soft-serve/server/config"
	"github.com/charmbracelet/soft-serve/server/git"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
)

// ErrServerClosed indicates that the server has been closed.
var ErrServerClosed = errors.New("git: Server closed")

// Daemon represents a Git daemon.
type Daemon struct {
	listener net.Listener
	addr     string
	finished chan struct{}
	conns    map[net.Conn]struct{}
	count    int
	cfg      *config.Config
	wg       sync.WaitGroup
	once     sync.Once
	mtx      sync.RWMutex
}

// NewDaemon returns a new Git daemon.
func NewDaemon(cfg *config.Config) (*Daemon, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Git.Port)
	d := &Daemon{
		addr:     addr,
		finished: make(chan struct{}, 1),
		cfg:      cfg,
		conns:    make(map[net.Conn]struct{}),
	}
	listener, err := net.Listen("tcp", d.addr)
	if err != nil {
		return nil, err
	}
	d.listener = listener
	return d, nil
}

// Start starts the Git TCP daemon.
func (d *Daemon) Start() error {
	defer d.listener.Close()

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
				log.Printf("git: error accepting connection: %v", err)
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
		var count int
		d.mtx.RLock()
		count = d.count
		d.mtx.RUnlock()
		log.Printf("count: %d", count)
		if count+1 >= d.cfg.Git.MaxConnections {
			log.Printf("git: max connections reached, closing %s", conn.RemoteAddr())
			fatal(conn, git.ErrMaxConnections)
			continue
		}

		d.wg.Add(1)
		go func() {
			d.handleClient(conn)
			d.wg.Done()
		}()
	}
}

func fatal(c net.Conn, err error) {
	git.WritePktline(c, err)
	if err := c.Close(); err != nil {
		log.Printf("git: error closing connection: %v", err)
	}
}

// handleClient handles a git protocol client.
func (d *Daemon) handleClient(conn net.Conn) {
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
	d.count++
	d.conns[c] = struct{}{}
	defer c.Close()
	defer func() {
		d.count--
		delete(d.conns, c)
	}()

	readc := make(chan struct{}, 1)
	s := pktline.NewScanner(c)
	go func() {
		if !s.Scan() {
			if err := s.Err(); err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					fatal(c, git.ErrTimeout)
				} else {
					log.Printf("git: error scanning pktline: %v", err)
					fatal(c, git.ErrSystemMalfunction)
				}
			}
			return
		}
		readc <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		if err := ctx.Err(); err != nil {
			log.Printf("git: connection context error: %v", err)
		}
		return
	case <-readc:
		line := s.Bytes()
		split := bytes.SplitN(line, []byte{' '}, 2)
		if len(split) != 2 {
			fatal(c, git.ErrInvalidRequest)
			return
		}

		var repo string
		cmd := string(split[0])
		opts := bytes.Split(split[1], []byte{'\x00'})
		if len(opts) == 0 {
			fatal(c, git.ErrInvalidRequest)
			return
		}
		repo = filepath.Clean(string(opts[0]))

		log.Printf("git: connect %s %s %s", c.RemoteAddr(), cmd, repo)
		defer log.Printf("git: disconnect %s %s %s", c.RemoteAddr(), cmd, repo)
		repo = strings.TrimPrefix(repo, "/")
		auth := d.cfg.AuthRepo(strings.TrimSuffix(repo, ".git"), nil)
		if auth < proto.ReadOnlyAccess {
			fatal(c, git.ErrNotAuthed)
			return
		}
		// git bare repositories should end in ".git"
		// https://git-scm.com/docs/gitrepository-layout
		if !strings.HasSuffix(repo, ".git") {
			repo += ".git"
		}

		err := git.GitPack(c, c, c, cmd, d.cfg.RepoPath(), repo)
		if err == git.ErrInvalidRepo {
			trimmed := strings.TrimSuffix(repo, ".git")
			log.Printf("git: invalid repo %q trying again %q", repo, trimmed)
			err = git.GitPack(c, c, c, cmd, d.cfg.RepoPath(), trimmed)
		}
		if err != nil {
			fatal(c, err)
			return
		}
	}
}

// Close closes the underlying listener.
func (d *Daemon) Close() error {
	d.once.Do(func() { close(d.finished) })
	err := d.listener.Close()
	for c := range d.conns {
		c.Close()
		delete(d.conns, c)
	}
	return err
}

// Shutdown gracefully shuts down the daemon.
func (d *Daemon) Shutdown(ctx context.Context) error {
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
