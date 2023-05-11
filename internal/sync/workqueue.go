package sync

import "sync"

// WorkQueue is a queue of work to be done.
type WorkQueue struct {
	work   map[string]func()
	wokers int
	mu     sync.RWMutex
	queue  chan string
}

// NewWorkQueue creates a new work queue. The workers argument specifies the
// number of concurrent workers to run the work.
// The queue will chunk the work into batches of workers size.
func NewWorkQueue(workers int) *WorkQueue {
	if workers <= 0 {
		workers = 1
	}

	return &WorkQueue{
		queue:  make(chan string),
		work:   make(map[string]func()),
		wokers: workers,
	}
}

// Run starts the workers and waits for them to finish.
func (wq *WorkQueue) Run() {
	for {
		wq.mu.RLock()
		work := len(wq.work)
		wq.mu.RUnlock()
		if work <= 0 {
			break
		}

		var wg sync.WaitGroup

		workers := wq.wokers
		if workers > work {
			workers = work
		}

		wg.Add(workers)
		for id, fn := range wq.work {
			if workers <= 0 {
				break
			}

			go func(id string, fn func()) {
				defer wg.Done()
				fn()
				wq.mu.Lock()
				delete(wq.work, id)
				wq.mu.Unlock()
			}(id, fn)

			workers--
		}

		wg.Wait()
	}
}

// Add adds a new job to the queue.
func (wq *WorkQueue) Add(id string, fn func()) {
	wq.mu.Lock()
	defer wq.mu.Unlock()
	if _, ok := wq.work[id]; ok {
		return
	}
	wq.work[id] = fn
}

// Status returns the Status of the given key.
func (wq *WorkQueue) Status(id string) bool {
	wq.mu.RLock()
	defer wq.mu.RUnlock()
	_, ok := wq.work[id]
	return ok
}
