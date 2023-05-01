package test

import "net"

// RandomPort returns a random port number.
// This is mainly used for testing.
func RandomPort() int {
	addr, _ := net.Listen("tcp", ":0") //nolint:gosec
	_ = addr.Close()
	return addr.Addr().(*net.TCPAddr).Port
}
