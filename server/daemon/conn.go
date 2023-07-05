package daemon

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"
)

// connections is a synchronizes access to to a net.Conn pool.
type connections struct {
	m  map[net.Conn]struct{}
	mu sync.Mutex
}

func (m *connections) Add(c net.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[c] = struct{}{}
}

func (m *connections) Close(c net.Conn) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	err := c.Close()
	delete(m.m, c)
	return err
}

func (m *connections) Size() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.m)
}

func (m *connections) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var err error
	for c := range m.m {
		err = errors.Join(err, c.Close())
		delete(m.m, c)
	}

	return err
}

// serverConn is a wrapper around a net.Conn that closes the connection when
// the one of the timeouts is reached.
type serverConn struct {
	net.Conn

	initTimeout   time.Duration
	idleTimeout   time.Duration
	maxDeadline   time.Time
	closeCanceler context.CancelFunc
}

var _ net.Conn = (*serverConn)(nil)

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
	case c.initTimeout > 0:
		initTimeout := time.Now().Add(c.initTimeout)
		c.initTimeout = 0
		if initTimeout.Unix() < c.maxDeadline.Unix() || c.maxDeadline.IsZero() {
			c.Conn.SetDeadline(initTimeout)
			return
		}
	case c.idleTimeout > 0:
		idleDeadline := time.Now().Add(c.idleTimeout)
		if idleDeadline.Unix() < c.maxDeadline.Unix() || c.maxDeadline.IsZero() {
			c.Conn.SetDeadline(idleDeadline)
			return
		}
	}
	c.Conn.SetDeadline(c.maxDeadline)
}
