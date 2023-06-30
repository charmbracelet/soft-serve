package logger

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/config"
)

// NewDefaultLogger returns a new logger with default settings.
func NewDefaultLogger(ctx context.Context) *log.Logger {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.DateOnly,
	})

	if debug, _ := strconv.ParseBool(os.Getenv("SOFT_SERVE_DEBUG")); debug {
		logger.SetLevel(log.DebugLevel)

		if verbose, _ := strconv.ParseBool(os.Getenv("SOFT_SERVE_VERBOSE")); verbose {
			logger.SetReportCaller(true)
		}
	}

	cfg := config.FromContext(ctx)
	logger.SetTimeFormat(cfg.Log.TimeFormat)

	switch strings.ToLower(cfg.Log.Format) {
	case "json":
		logger.SetFormatter(log.JSONFormatter)
	case "logfmt":
		logger.SetFormatter(log.LogfmtFormatter)
	case "text":
		logger.SetFormatter(log.TextFormatter)
	}

	return logger
}
