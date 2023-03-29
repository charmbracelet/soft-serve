// Package log initializes the logger for Soft Serve modules.
package log

import (
	"os"

	"github.com/charmbracelet/log"
)

func init() {
	if os.Getenv("SOFT_SERVE_DEBUG") == "true" {
		log.SetLevel(log.DebugLevel)
	}
}
