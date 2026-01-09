package git

import (
	"charm.land/log/v2"
	"github.com/charmbracelet/git-lfs-transfer/transfer"
)

type lfsLogger struct {
	l *log.Logger
}

var _ transfer.Logger = &lfsLogger{}

// Log implements transfer.Logger.
func (l *lfsLogger) Log(msg string, kv ...interface{}) {
	l.l.Debug(msg, kv...)
}
