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
func (m *Manager) Run(id string, done chan<- error) {
	v, ok := m.m.Load(id)
	if !ok {
		done <- ErrNotFound
		return
	}

	p := v.(*Task)
	if p.started.Load() {
		<-p.ctx.Done()
		p.mu.Lock()
		err := p.err
		p.mu.Unlock()
		if err != nil {
			done <- err
			return
		}

		done <- p.ctx.Err()
		return
	}

	p.started.Store(true)
	m.m.Store(id, p)
	defer p.cancel()
	defer m.m.Delete(id)

	errc := make(chan error, 1)
	go func(ctx context.Context) {
		errc <- p.fn(ctx)
	}(p.ctx)

	select {
	case <-m.ctx.Done():
		done <- m.ctx.Err()
	case err := <-errc:
		p.mu.Lock()
		p.err = err
		p.mu.Unlock()
		m.m.Store(id, p)
		done <- err
	}
}
