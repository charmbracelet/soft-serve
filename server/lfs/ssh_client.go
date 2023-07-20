package lfs

import "github.com/charmbracelet/ssh"

// TODO: implement Git LFS SSH client.

type sshClient struct {
	s ssh.Session
}
