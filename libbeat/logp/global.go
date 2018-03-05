package logp

import (
	"fmt"

	"go.uber.org/zap"
)

// MakeDebug returns a function that logs at debug level.
// Deprecated: Use logp.NewLogger.
func MakeDebug(selector string) func(string, ...interface{}) {
	return func(format string, v ...interface{}) {
		globalLogger().Named(selector).Debug(fmt.Sprintf(format, v...))
	}
}

// HasSelector returns true if the given selector was explicitly set.
func HasSelector(selector string) bool {
	_, found := loadLogger().selectors[selector]
	return found
}

// IsDebug returns true if the given selector would be logged.
// Deprecated: Use logp.NewLogger.
func IsDebug(selector string) bool {
	return globalLogger().Named(selector).Check(zap.DebugLevel, "") != nil
}

// Debug uses fmt.Sprintf to construct and log a message.
// Deprecated: Use logp.NewLogger.
func Debug(selector string, format string, v ...interface{}) {
	log := globalLogger()
	if log.Core().Enabled(zap.DebugLevel) {
		log.Named(selector).Debug(fmt.Sprintf(format, v...))
	}
}

// Info uses fmt.Sprintf to construct and log a message.
// Deprecated: Use logp.NewLogger.
func Info(format string, v ...interface{}) {
	log := globalLogger()
	if log.Core().Enabled(zap.InfoLevel) {
		log.Info(fmt.Sprintf(format, v...))
	}
}

// Warn uses fmt.Sprintf to construct and log a message.
// Deprecated: Use logp.NewLogger.
func Warn(format string, v ...interface{}) {
	log := globalLogger()
	if log.Core().Enabled(zap.WarnLevel) {
		globalLogger().Warn(fmt.Sprintf(format, v...))
	}
}

// Err uses fmt.Sprintf to construct and log a message.
// Deprecated: Use logp.NewLogger.
func Err(format string, v ...interface{}) {
	log := globalLogger()
	if log.Core().Enabled(zap.ErrorLevel) {
		globalLogger().Error(fmt.Sprintf(format, v...))
	}
}

// Critical uses fmt.Sprintf to construct and log a message. It's an alias for
// Error.
// Deprecated: Use logp.NewLogger.
func Critical(format string, v ...interface{}) {
	log := globalLogger()
	if log.Core().Enabled(zap.ErrorLevel) {
		globalLogger().Error(fmt.Sprintf(format, v...))
	}
}

// WTF prints the message at PanicLevel and immediately panics with the same
// message.
//
// Deprecated: Use logp.NewLogger and its Panic or DPanic methods.
func WTF(format string, v ...interface{}) {
	globalLogger().Panic(fmt.Sprintf(format, v...))
}

// Recover stops a panicking goroutine and logs an Error.
func Recover(msg string) {
	if r := recover(); r != nil {
		msg := fmt.Sprintf("%s. Recovering, but please report this.", msg)
		globalLogger().WithOptions(zap.AddCallerSkip(2)).
			Error(msg, zap.Any("panic", r), zap.Stack("stack"))
	}
}
