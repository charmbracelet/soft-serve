package ssh

import "github.com/charmbracelet/log"

type loggerAdapter struct {
	*log.Logger
	log.Level
}

func (l *loggerAdapter) Printf(format string, args ...interface{}) {
	switch l.Level {
	case log.DebugLevel:
		l.Logger.Debugf(format, args...)
	case log.InfoLevel:
		l.Logger.Infof(format, args...)
	case log.WarnLevel:
		l.Logger.Warnf(format, args...)
	case log.ErrorLevel:
		l.Logger.Errorf(format, args...)
	case log.FatalLevel:
		l.Logger.Fatalf(format, args...)
	default:
		l.Logger.Printf(format, args...)
	}
}
