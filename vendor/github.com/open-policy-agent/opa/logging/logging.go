package logging

import (
	"io"

	"github.com/sirupsen/logrus"
)

// Level log level for Logger
type Level uint8

const (
	// Error error log level
	Error Level = iota
	// Warn warn log level
	Warn
	// Info info log level
	Info
	// Debug debug log level
	Debug
)

// Logger provides interface for OPA logger implementations
type Logger interface {
	Debug(fmt string, a ...interface{})
	Info(fmt string, a ...interface{})
	Error(fmt string, a ...interface{})
	Warn(fmt string, a ...interface{})

	WithFields(map[string]interface{}) Logger
	GetFields() map[string]interface{}

	GetLevel() Level
	SetLevel(Level)
}

// StandardLogger is the default OPA logger
type StandardLogger struct {
	logger *logrus.Logger
	fields map[string]interface{}
}

// New returns a new standard logger.
func New() *StandardLogger {
	return &StandardLogger{
		logger: logrus.New(),
	}
}

// NewStandardLogger is deprecated. Use Get instead.
func NewStandardLogger() *StandardLogger {
	return Get()
}

// Get returns the standard logger used throughout OPA.
func Get() *StandardLogger {
	return &StandardLogger{
		logger: logrus.StandardLogger(),
	}
}

// SetOutput sets the underlying logrus output.
func (l *StandardLogger) SetOutput(w io.Writer) {
	l.logger.SetOutput(w)
}

// SetFormatter sets the underlying logrus formatter.
func (l *StandardLogger) SetFormatter(formatter logrus.Formatter) {
	l.logger.SetFormatter(formatter)
}

// WithFields provides additional fields to include in log output
func (l *StandardLogger) WithFields(fields map[string]interface{}) Logger {
	cp := *l
	cp.fields = make(map[string]interface{})
	for k, v := range l.fields {
		cp.fields[k] = v
	}
	for k, v := range fields {
		cp.fields[k] = v
	}
	return &cp
}

// GetFields returns additional fields of this logger
func (l *StandardLogger) GetFields() map[string]interface{} {
	return l.fields
}

// SetLevel sets the standard logger level.
func (l *StandardLogger) SetLevel(level Level) {
	var logrusLevel logrus.Level
	switch level {
	case Error:
		logrusLevel = logrus.ErrorLevel
	case Warn:
		logrusLevel = logrus.WarnLevel
	case Info:
		logrusLevel = logrus.InfoLevel
	case Debug:
		logrusLevel = logrus.DebugLevel
	default:
		l.Warn("unknown log level %v", level)
		logrusLevel = logrus.InfoLevel
	}

	l.logger.SetLevel(logrusLevel)
}

// GetLevel returns the standard logger level.
func (l *StandardLogger) GetLevel() Level {
	logrusLevel := l.logger.GetLevel()

	var level Level
	switch logrusLevel {
	case logrus.ErrorLevel:
		level = Error
	case logrus.WarnLevel:
		level = Warn
	case logrus.InfoLevel:
		level = Info
	case logrus.DebugLevel:
		level = Debug
	default:
		l.Warn("unknown log level %v", logrusLevel)
		level = Info
	}

	return level
}

// Debug logs at debug level
func (l *StandardLogger) Debug(fmt string, a ...interface{}) {
	l.logger.WithFields(l.GetFields()).Debugf(fmt, a...)
}

// Info logs at info level
func (l *StandardLogger) Info(fmt string, a ...interface{}) {
	l.logger.WithFields(l.GetFields()).Infof(fmt, a...)
}

// Error logs at error level
func (l *StandardLogger) Error(fmt string, a ...interface{}) {
	l.logger.WithFields(l.GetFields()).Errorf(fmt, a...)
}

// Warn logs at warn level
func (l *StandardLogger) Warn(fmt string, a ...interface{}) {
	l.logger.WithFields(l.GetFields()).Errorf(fmt, a...)
}

// NoOpLogger logging implementation that does nothing
type NoOpLogger struct {
	level  Level
	fields map[string]interface{}
}

// NewNoOpLogger instantiates new NoOpLogger
func NewNoOpLogger() *NoOpLogger {
	return &NoOpLogger{
		level: Info,
	}
}

// WithFields provides additional fields to include in log output.
// Implemented here primarily to be able to switch between implementations without loss of data.
func (l *NoOpLogger) WithFields(fields map[string]interface{}) Logger {
	cp := *l
	cp.fields = fields
	return &cp
}

// GetFields returns additional fields of this logger
// Implemented here primarily to be able to switch between implementations without loss of data.
func (l *NoOpLogger) GetFields() map[string]interface{} {
	return l.fields
}

// Debug noop
func (*NoOpLogger) Debug(string, ...interface{}) {}

// Info noop
func (*NoOpLogger) Info(string, ...interface{}) {}

// Error noop
func (*NoOpLogger) Error(string, ...interface{}) {}

// Warn noop
func (*NoOpLogger) Warn(string, ...interface{}) {}

// SetLevel set log level
func (l *NoOpLogger) SetLevel(level Level) {
	l.level = level
}

// GetLevel get log level
func (l *NoOpLogger) GetLevel() Level {
	return l.level
}
