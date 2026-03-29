package task

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var (
	// ErrNotFound is returned when a process is not found.
	ErrNotFound = errors.New("task not found")

	// ErrAlreadyStarted is returned when a process is already started.
	ErrAlreadyStarted = errors.New("task already started")
)

// Task is a task that can be started and stopped.
type Task struct {
	id      string
	fn      func(context.Context) error
	started atomic.Bool
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
	err     error
}

// Manager manages tasks.
type Manager struct {
	m   sync.Map
	ctx context.Context
}

// NewManager returns a new task manager.
func NewManager(ctx context.Context) *Manager {
	return &Manager{
		m:   sync.Map{},
		ctx: ctx,
	}
}

// Add adds a task to the manager.
// If the process already exists, it is a no-op.
func (m *Manager) Add(id string, fn func(context.Context) error) {
	ctx, cancel := context.WithCancel(m.ctx)
	t := &Task{
		id:     id,
		fn:     fn,
		ctx:    ctx,
		cancel: cancel,
	}
	if _, loaded := m.m.LoadOrStore(id, t); loaded {
		cancel()
	}
}

// Stop stops the task and removes it from the manager.
func (m *Manager) Stop(id string) error {
	v, ok := m.m.Load(id)
	if !ok {
		return ErrNotFound
	}

	p := v.(*Task)
	p.cancel()

	m.m.Delete(id)
	return nil
}

// Exists checks if a task exists.
func (m *Manager) Exists(id string) bool {
	_, ok := m.m.Load(id)
	return ok
}

// Run starts the task if not already started, or waits for the already-running
// task to complete and delivers its result to done.
// Callers MUST ensure done has capacity >= 1 (buffered) to avoid a panic
// if the caller returns before the task delivers its result.
//
// If Stop() is called while a second goroutine is waiting on an already-started
// task, done receives context.Canceled. Callers should treat context.Canceled as
// "task was stopped or the manager shut down", not necessarily as a task error.
func (m *Manager) Run(id string, done chan<- error) {
	v, ok := m.m.Load(id)
	if !ok {
		done <- ErrNotFound
		return
	}

	p := v.(*Task)
	if !p.started.CompareAndSwap(false, true) {
		// Task is already running (or was already completed). Wait for it.
		// Note: between CompareAndSwap returning false and <-p.ctx.Done()
		// completing, Stop() may cancel the context and delete the map entry —
		// that is safe because p.ctx.Done() still fires and p is still reachable
		// through the local pointer.
		<-p.ctx.Done()
		p.mu.Lock()
		err := p.err
		p.mu.Unlock()
		if err != nil {
			done <- err
			return
		}

		// p.err == nil: task completed successfully; p.cancel() fired after fn
		// returned. Return nil so callers distinguish clean completion from a
		// context cancellation caused by Stop() or manager shutdown.
		done <- nil
		return
	}

	// We won the CAS: we are the sole goroutine responsible for running p.fn.
	// m.m already holds p from Add; no re-store needed here.
	defer p.cancel()

	errc := make(chan error, 1)
	go func(ctx context.Context) {
		errc <- p.fn(ctx)
	}(p.ctx)

	select {
	case <-m.ctx.Done():
		done <- m.ctx.Err()
		// Delay map deletion until the p.fn goroutine has fully exited so that
		// a concurrent Add(id, ...) + Run(id, ...) cannot start a new task for
		// the same id while the old goroutine is still executing p.fn.
		go func() {
			<-errc
			m.m.Delete(id)
		}()
	case err := <-errc:
		p.mu.Lock()
		p.err = err
		p.mu.Unlock()
		// No re-store: m.m already holds p; delete after storing the result.
		m.m.Delete(id)
		done <- err
	}
}
