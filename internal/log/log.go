package log

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

var contextKey = &struct{ string }{"logger"}

// NewDefaultLogger returns a new logger with default settings.
func NewDefaultLogger() *log.Logger {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.DateOnly,
	})

	if debug, _ := strconv.ParseBool(os.Getenv("SOFT_SERVE_DEBUG")); debug {
		logger.SetLevel(log.DebugLevel)
	}

	if tsfmt := os.Getenv("SOFT_SERVE_LOG_TIME_FORMAT"); tsfmt != "" {
		logger.SetTimeFormat(tsfmt)
	}

	switch strings.ToLower(os.Getenv("SOFT_SERVE_LOG_FORMAT")) {
	case "json":
		logger.SetFormatter(log.JSONFormatter)
	case "logfmt":
		logger.SetFormatter(log.LogfmtFormatter)
	case "text":
		logger.SetFormatter(log.TextFormatter)
	}

	return logger
}
