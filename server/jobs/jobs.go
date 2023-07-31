package jobs

import (
	"context"
	"sync"
)

// Job is a job that can be registered with the scheduler.
type Job struct {
	ID   int
	Spec string
	Func func(context.Context) func()
}

var (
	mtx  sync.Mutex
	jobs = make(map[string]*Job, 0)
)

// Register registers a job.
func Register(name, spec string, fn func(context.Context) func()) {
	mtx.Lock()
	defer mtx.Unlock()
	jobs[name] = &Job{Spec: spec, Func: fn}
}

// List returns a map of registered jobs.
func List() map[string]*Job {
	mtx.Lock()
	defer mtx.Unlock()
	return jobs
}
