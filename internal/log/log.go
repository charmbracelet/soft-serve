package log

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/config"
)

var contextKey = &struct{ string }{"logger"}

// NewDefaultLogger returns a new logger with default settings.
func NewDefaultLogger() *log.Logger {
	dp := os.Getenv("SOFT_SERVE_DATA_PATH")
	if dp == "" {
		dp = "data"
	}

	cfg, err := config.ParseConfig(filepath.Join(dp, "config.yaml"))
	if err != nil {
		log.Errorf("failed to parse config: %v", err)
	}

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
