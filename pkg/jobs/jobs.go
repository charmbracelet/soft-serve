// Package jobs provides background job processing functionality.
package jobs

import (
	"context"
	"sync"
)

// Job is a job that can be registered with the scheduler.
type Job struct {
	ID     int
	Runner Runner
}

// Runner is a job runner.
type Runner interface {
	Spec(context.Context) string
	Func(context.Context) func()
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
