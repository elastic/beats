package logp

import "fmt"

// Logger provides a logging type using the global logp functionality.
// The Logger should be used to use with libraries havng a configurable logging
// functionality.
type Logger struct {
	selector string
}

// NewLogger creates a new Logger instance with custom debug selector.
func NewLogger(selector string) *Logger {
	return &Logger{selector: selector}
}

func (l *Logger) Debug(vs ...interface{}) {
	Debug(l.selector, "%v", fmt.Sprint(vs...))
}

func (*Logger) Info(vs ...interface{}) {
	Info("%v", fmt.Sprint(vs...))
}

func (*Logger) Err(vs ...interface{}) {
	Err("%v", fmt.Sprint(vs...))
}

func (l *Logger) Debugf(format string, v ...interface{}) { Debug(l.selector, format, v...) }
func (*Logger) Infof(format string, v ...interface{})    { Info(format, v...) }
func (*Logger) Errf(format string, v ...interface{})     { Err(format, v...) }
