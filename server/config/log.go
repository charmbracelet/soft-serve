package config

import "time"

// Log is the logger configuration.
var Log = &LogConfig{
	Format:     "text",
	TimeFormat: time.DateTime,
}

// LogConfig is the logger configuration.
type LogConfig struct {
	// Format is the format of the logs.
	// Valid values are "json", "logfmt", and "text".
	Format string `env:"FORMAT" yaml:"format"`

	// Time format for the log `ts` field.
	// Format must be described in Golang's time format.
	TimeFormat string `env:"TIME_FORMAT" yaml:"time_format"`
}

// Environ returns the environment variables for the config.
func (l LogConfig) Environ() []string {
	return []string{
		"SOFT_SERVE_LOG_FORMAT=" + l.Format,
		"SOFT_SERVE_LOG_TIME_FORMAT=" + l.TimeFormat,
	}
}
