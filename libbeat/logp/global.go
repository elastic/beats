package logp

import (
	"fmt"

	"go.uber.org/zap"
)

// Deprecated: Use logp.NewLogger or logp.NewSimpleLogger.
func MakeDebug(selector string) func(string, ...interface{}) {
	return func(format string, v ...interface{}) {
		globalLogger().Named(selector).Debug(fmt.Sprintf(format, v...))
	}
}

// Deprecated: Use logp.NewLogger or logp.NewSimpleLogger.
func IsDebug(selector string) bool {
	return globalLogger().Named(selector).Check(zap.DebugLevel, "") != nil
}

// Deprecated: Use logp.NewLogger or logp.NewSimpleLogger.
func Debug(selector string, format string, v ...interface{}) {
	globalLogger().Named(selector).Debug(fmt.Sprintf(format, v...))
}

// Deprecated: Use logp.NewLogger or logp.NewSimpleLogger.
func Info(format string, v ...interface{}) {
	globalLogger().Info(fmt.Sprintf(format, v...))
}

// Deprecated: Use logp.NewLogger or logp.NewSimpleLogger.
func Warn(format string, v ...interface{}) {
	globalLogger().Warn(fmt.Sprintf(format, v...))
}

// Deprecated: Use logp.NewLogger or logp.NewSimpleLogger.
func Err(format string, v ...interface{}) {
	globalLogger().Error(fmt.Sprintf(format, v...))
}

// Deprecated: Use logp.NewLogger or logp.NewSimpleLogger.
func Critical(format string, v ...interface{}) {
	globalLogger().Fatal(fmt.Sprintf(format, v...))
}

// WTF prints the message at CRIT level and panics immediately with the same
// message
//
// Deprecated: Use logp.NewLogger or logp.NewSimpleLogger.
func WTF(format string, v ...interface{}) {
	globalLogger().Panic(fmt.Sprintf(format, v...))
}

func Recover(msg string) {
	if r := recover(); r != nil {
		globalLogger().Error("Recovering, but please report this",
			zap.Any("panic", msg), zap.Stack("stack"))
	}
}

type Priority int

const (
	// From /usr/include/sys/syslog.h.
	// These are the same on Linux, BSD, and OS X.
	LOG_EMERG Priority = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

func LogInit(level Priority, prefix string, toSyslog bool, toStderr bool, debugSelectors []string) {
	// TODO: Incorporate selectors and level into dev setup.
	// TODO: Remove all usages of LogInit.
	DevelopmentSetup()
}
