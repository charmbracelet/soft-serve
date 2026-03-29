package task

import (
	"context"
	"errors"
	"fmt"
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
	id        string
	fn        func(context.Context) error
	started   atomic.Bool
	// completed must be stored only AFTER p.err is written and p.mu is
	// unlocked. A concurrent waiter observing completed==true via its own
	// p.mu.Lock() is then guaranteed to see the final value of p.err.
	// Never move this Store before p.mu.Unlock() or the ordering guarantee breaks.
	completed atomic.Bool
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	err       error
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
		// Block until the task's context is done. The winner calls p.cancel()
		// via defer AFTER writing p.err and storing p.completed=true, so by
		// the time <-p.ctx.Done() returns, both writes are visible (Go memory
		// model: the defer executes after the preceding statements, providing
		// happens-before from the stores to the channel close).
		<-p.ctx.Done()

		// Check completed first: if true, the task finished normally and p.err
		// holds the authoritative result (nil or non-nil). We must read p.err
		// under the mutex to synchronise with the write in the winner path.
		if p.completed.Load() {
			p.mu.Lock()
			err := p.err
			p.mu.Unlock()
			done <- err
			return
		}

		// completed is false: p.cancel() was called by Stop() or the manager
		// shutting down before the task finished. Return the context error so
		// callers can distinguish a premature stop from a task error.
		done <- p.ctx.Err()
		return
	}

	// We won the CAS: we are the sole goroutine responsible for running p.fn.
	// m.m already holds p from Add; no re-store needed here.
	defer p.cancel()

	errc := make(chan error, 1)
	go func(ctx context.Context) {
		defer func() {
			if r := recover(); r != nil {
				errc <- fmt.Errorf("task panicked: %v", r)
			}
		}()
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
		// Mark completed BEFORE p.cancel() fires (via defer) so that any
		// concurrent waiter on <-p.ctx.Done() sees completed=true and
		// returns nil rather than ctx.Err() for a successfully-finished task.
		p.completed.Store(true)
		// Deliver the result first, then remove from map. A concurrent
		// Run() arriving between Delete and done<-err would get ErrNotFound;
		// by deleting after the send, any such caller simply misses this task
		// (it has already completed) rather than observing an inconsistent state.
		done <- err
		m.m.Delete(id)
	}
}
