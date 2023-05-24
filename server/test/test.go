package test

import (
	"net"
	"sync"
)

var (
	used = map[int]struct{}{}
	lock sync.Mutex
)

// RandomPort returns a random port number.
// This is mainly used for testing.
func RandomPort() int {
	addr, _ := net.Listen("tcp", ":0") //nolint:gosec
	_ = addr.Close()
	port := addr.Addr().(*net.TCPAddr).Port
	lock.Lock()

	if _, ok := used[port]; ok {
		lock.Unlock()
		return RandomPort()
	}

	used[port] = struct{}{}
	lock.Unlock()
	return port
}
