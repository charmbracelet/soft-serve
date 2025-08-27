package cron

import (
	"context"
	"time"

	log "github.com/charmbracelet/log/v2"
	"github.com/robfig/cron/v3"
)

// Scheduler is a cron-like job scheduler.
type Scheduler struct {
	*cron.Cron
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

// NewScheduler returns a new Cron.
func NewScheduler(ctx context.Context) *Scheduler {
	logger := cronLogger{log.FromContext(ctx).WithPrefix("cron")}
	return &Scheduler{
		Cron: cron.New(cron.WithLogger(logger)),
	}
}

// Shutdown gracefully shuts down the Scheduler.
func (s *Scheduler) Shutdown() {
	ctx, cancel := context.WithTimeout(s.Cron.Stop(), 30*time.Second)
	defer func() { cancel() }()
	<-ctx.Done()
}

// Start starts the Scheduler.
func (s *Scheduler) Start() {
	s.Cron.Start()
}

// AddFunc adds a job to the Scheduler.
func (s *Scheduler) AddFunc(spec string, fn func()) (int, error) {
	id, err := s.Cron.AddFunc(spec, fn)
	return int(id), err
}

// Remove removes a job from the Scheduler.
func (s *Scheduler) Remove(id int) {
	s.Cron.Remove(cron.EntryID(id))
}
