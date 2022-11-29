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
	exit     chan struct{}
	conns    map[net.Conn]struct{}
	cfg      *config.Config
	wg       sync.WaitGroup
	once     sync.Once
}

// NewDaemon returns a new Git daemon.
func NewDaemon(cfg *config.Config) (*Daemon, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Git.Port)
	d := &Daemon{
		addr:  addr,
		exit:  make(chan struct{}),
		cfg:   cfg,
		conns: make(map[net.Conn]struct{}),
	}
	listener, err := net.Listen("tcp", d.addr)
	if err != nil {
		return nil, err
	}
	d.listener = listener
	d.wg.Add(1)
	return d, nil
}

// Start starts the Git TCP daemon.
func (d *Daemon) Start() error {
	// set up channel on which to send accepted connections
	listen := make(chan net.Conn, d.cfg.Git.MaxConnections)
	go d.acceptConnection(d.listener, listen)

	// loop work cycle with accept connections or interrupt
	// by system signal
	for {
		select {
		case conn := <-listen:
			d.wg.Add(1)
			go func() {
				d.handleClient(conn)
				d.wg.Done()
			}()
		case <-d.exit:
			if err := d.Close(); err != nil {
				return err
			}
			return ErrServerClosed
		}
	}
}

func fatal(c net.Conn, err error) {
	git.WritePktline(c, err)
	if err := c.Close(); err != nil {
		log.Printf("git: error closing connection: %v", err)
	}
}

// acceptConnection accepts connections on the listener.
func (d *Daemon) acceptConnection(listener net.Listener, listen chan<- net.Conn) {
	defer d.wg.Done()
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-d.exit:
				log.Printf("git: listener closed")
				return
			default:
				log.Printf("git: error accepting connection: %v", err)
				continue
			}
		}
		listen <- conn
	}
}

// handleClient handles a git protocol client.
func (d *Daemon) handleClient(c net.Conn) {
	d.conns[c] = struct{}{}
	defer delete(d.conns, c)

	// Close connection if there are too many open connections.
	if len(d.conns) >= d.cfg.Git.MaxConnections {
		log.Printf("git: max connections reached, closing %s", c.RemoteAddr())
		fatal(c, git.ErrMaxConns)
		return
	}

	// Set connection timeout.
	if err := c.SetDeadline(time.Now().Add(time.Duration(d.cfg.Git.MaxTimeout) * time.Second)); err != nil {
		log.Printf("git: error setting deadline: %v", err)
		fatal(c, git.ErrSystemMalfunction)
		return
	}

	readc := make(chan struct{}, 1)
	go func() {
		select {
		case <-time.After(time.Duration(d.cfg.Git.MaxReadTimeout) * time.Second):
			log.Printf("git: read timeout from %s", c.RemoteAddr())
			fatal(c, git.ErrMaxTimeout)
		case <-readc:
		}
	}()

	s := pktline.NewScanner(c)
	if !s.Scan() {
		if err := s.Err(); err != nil {
			log.Printf("git: error scanning pktline: %v", err)
			fatal(c, git.ErrSystemMalfunction)
		}
		return
	}
	readc <- struct{}{}

	line := s.Bytes()
	split := bytes.SplitN(line, []byte{' '}, 2)
	if len(split) != 2 {
		return
	}

	var repo string
	cmd := string(split[0])
	opts := bytes.Split(split[1], []byte{'\x00'})
	if len(opts) == 0 {
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

// Close closes the underlying listener.
func (d *Daemon) Close() error {
	d.once.Do(func() { close(d.exit) })
	return d.listener.Close()
}

// Shutdown gracefully shuts down the daemon.
func (d *Daemon) Shutdown(_ context.Context) error {
	d.once.Do(func() { close(d.exit) })
	d.wg.Wait()
	return nil
}
