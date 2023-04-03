package cron

import (
	"context"
	"time"

	"github.com/charmbracelet/log"
	"github.com/robfig/cron/v3"
)

// CronScheduler is a cron-like job scheduler.
type CronScheduler struct {
	*cron.Cron
	logger cron.Logger
}

// Entry is a cron job.
type Entry struct {
	ID   cron.EntryID
	Desc string
	Spec string
}

// cronLogger is a wrapper around the logger to make it compatible with the
// cron logger.
type cronLogger struct {
	logger *log.Logger
}

// Info logs routine messages about cron's operation.
func (l cronLogger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Debug(msg, keysAndValues...)
}

// Error logs an error condition.
func (l cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.logger.Error(msg, append(keysAndValues, "err", err)...)
}

// NewCronScheduler returns a new Cron.
func NewCronScheduler() *CronScheduler {
	logger := cronLogger{log.WithPrefix("server.cron")}
	return &CronScheduler{
		Cron: cron.New(cron.WithLogger(logger)),
	}
}

// Shutdonw gracefully shuts down the CronServer.
func (s *CronScheduler) Shutdown() {
	ctx, cancel := context.WithTimeout(s.Cron.Stop(), 30*time.Second)
	defer func() { cancel() }()
	<-ctx.Done()
}

// Start starts the CronServer.
func (s *CronScheduler) Start() {
	s.Cron.Start()
}
