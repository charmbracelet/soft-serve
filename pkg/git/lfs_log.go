package git

import (
	"github.com/charmbracelet/git-lfs-transfer/transfer"
	"github.com/charmbracelet/log"
)

type lfsLogger struct {
	l *log.Logger
}

var _ transfer.Logger = &lfsLogger{}

// Log implements transfer.Logger.
func (l *lfsLogger) Log(msg string, kv ...interface{}) {
	l.l.Debug(msg, kv...)
}
