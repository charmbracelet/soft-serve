package log

import (
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/charmbracelet/soft-serve/server/config"
)

// NewLogger returns a new logger with default settings.
func NewLogger(cfg *config.Config) (*log.Logger, *os.File, error) {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		TimeFormat:      time.DateOnly,
	})

	switch {
	case config.IsVerbose():
		logger.SetReportCaller(true)
		fallthrough
	case config.IsDebug():
		logger.SetLevel(log.DebugLevel)
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

	var f *os.File
	if cfg.Log.Path != "" {
		f, err := os.OpenFile(cfg.Log.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, err
		}
		logger.SetOutput(f)
	}

	return logger, f, nil
}
