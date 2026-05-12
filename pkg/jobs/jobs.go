package jobs

import (
	"context"
	"io"
	"sync"

	"github.com/spf13/pflag"
)

// Job is a job that can be registered with the scheduler.
type Job struct {
	ID     int
	Runner Runner
}

// Runner is a job runner.
type Runner interface {
	Description() string
	Config(context.Context) (RunnerConfig, error)

	Func(context.Context, RunnerConfig) func()
}

// JobConfig is the configuration for Runner, passed to Runner.Func
type RunnerConfig interface {
	FlagSet() *pflag.FlagSet
	Spec() string

	SetOut(out io.Writer)
	Output() io.Writer
	SetErr(err io.Writer)
	Error() io.Writer
}

// baseRunnerConfig implements the common part of job tasks' RunnerConfig
type baseRunnerConfig struct {
	CronSpec string `yaml:"spec"`

	output io.Writer
	error  io.Writer
}

// SetOut sets the stdout of cron job
func (cfg *baseRunnerConfig) SetOut(out io.Writer) { cfg.output = out }

// Output return the stdout of cron job
func (cfg *baseRunnerConfig) Output() io.Writer {
	if cfg.output == nil {
		return io.Discard
	}
	return cfg.output
}

// SetErr sets the stderr of cron job
func (cfg *baseRunnerConfig) SetErr(err io.Writer) { cfg.error = err }

// Error return the stderr of cron job
func (cfg *baseRunnerConfig) Error() io.Writer {
	if cfg.error == nil {
		return io.Discard
	}
	return cfg.error
}

// Spec derives the spec for built-in job scheduler and implements RunnerConfig.
func (cfg *baseRunnerConfig) Spec() string {
	return cfg.CronSpec
}

var (
	mtx  sync.Mutex
	jobs = make(map[string]*Job, 0)
)

// Register registers a job.
func Register(name string, runner Runner) {
	mtx.Lock()
	defer mtx.Unlock()
	jobs[name] = &Job{Runner: runner}
}

// List returns a map of registered jobs.
func List() map[string]*Job {
	mtx.Lock()
	defer mtx.Unlock()
	return jobs
}
