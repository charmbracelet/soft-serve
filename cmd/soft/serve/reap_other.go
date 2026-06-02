//go:build !linux

package serve

import (
	"context"

	"charm.land/log/v2"
)

// reapZombies is a no-op on non-Linux platforms.
// See reap_linux.go for the Linux implementation.
func reapZombies(_ context.Context, _ *log.Logger) {}
