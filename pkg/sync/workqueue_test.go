package sync

import (
	"context"
	"strconv"
	"sync"
	"testing"
)

func TestWorkPool(t *testing.T) {
	mtx := &sync.Mutex{}
	values := make([]int, 0)
	wp := NewWorkPool(context.Background(), 3)
	for i := 0; i < 10; i++ {
		id := strconv.Itoa(i)
		i := i
		wp.Add(id, func() {
			mtx.Lock()
			values = append(values, i)
			mtx.Unlock()
		})
	}
	wp.Run()

	if len(values) != 10 {
		t.Errorf("expected 10 values, got %d, %v", len(values), values)
	}

	for i := range values {
		id := strconv.Itoa(i)
		if wp.Status(id) {
			t.Errorf("expected %s to be false", id)
		}
	}
}
