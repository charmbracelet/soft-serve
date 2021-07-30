package middleware

import "github.com/gliderlabs/ssh"

type Middleware func(ssh.Handler) ssh.Handler
