//go:build linux

package serve

import (
	"context"
	"time"

	"charm.land/log/v2"
	"golang.org/x/sys/unix"
)

// reapZombies periodically reaps zombie child processes.
//
// When soft-serve runs as PID 1 in a container, orphaned descendant
// processes (e.g. grandchild git pack-objects left behind when a git
// parent exits before waiting for them) are reparented to PID 1.
// Without an init system these processes become zombies because the
// Go runtime only tracks children spawned via os/exec, not reparented
// orphans. This goroutine periodically calls waitpid(-1, WNOHANG) to
// clean them up.
func reapZombies(ctx context.Context, logger *log.Logger) {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for {
					var status unix.WaitStatus
					pid, err := unix.Wait4(-1, &status, unix.WNOHANG, nil)
					if err != nil || pid <= 0 {
						break
					}
					logger.Debugf("reaped zombie child pid=%d status=%d", pid, status)
				}
			}
		}
	}()
}
