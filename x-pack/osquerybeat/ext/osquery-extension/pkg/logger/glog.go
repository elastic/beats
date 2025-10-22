// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package logger provides a lightweight glog-compatible logger for osquery extensions.
// The logger outputs messages in the same format as osqueryd (Google glog format),
// matches the log level used by osqueryd (warnings/errors by default, info with --verbose),
// and enables debug logging when the --verbose flag is passed.
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/osquery/osquery-go/gen/osquery"
)

// Following golang/glog's approach, we use the process ID since thread IDs
// are not meaningful in Go due to goroutine multiplexing across OS threads.
var pid = os.Getpid()

// osExit is a variable to allow mocking os.Exit in tests
var osExit = os.Exit

// Level represents the log severity level matching osquery's glog levels
type Level int

const (
	// LevelInfo represents informational messages
	LevelInfo Level = iota
	// LevelWarning represents warning messages
	LevelWarning
	// LevelError represents error messages
	LevelError
)

// ParseLogLevel parses osquery's logger_min_status flag value
// Osquery uses numeric values: 0=INFO, 1=WARNING, 2=ERROR
func parseLogLevelOption(opt osquery.InternalOptionInfo) Level {
	v, err := strconv.ParseInt(opt.Value, 10, 64)
	if err != nil {
		return LevelWarning
	}
	return Level(v)
}

// ParseBool parses string boolean values (true/false, 1/0, yes/no)
func parseUTCOption(opt osquery.InternalOptionInfo) bool {
	switch opt.Value {
	case "true", "1", "yes", "TRUE", "YES":
		return true
	case "false", "0", "no", "FALSE", "NO":
		return false
	default:
		return true
	}
}

// Logger writes logs in Google glog format to match osqueryd output
// Format: I0314 15:24:36.123456 12345 file.cpp:123] message
type Logger struct {
	verbose  bool
	mu       sync.Mutex
	w        io.Writer
	minLevel Level
	useUTC   bool
}

// New creates a new glog-formatted logger with default configuration
// Set verbose=true to enable debug/info level logging
func New(w io.Writer, verbose bool) *Logger {
	minLevel := LevelWarning
	if verbose {
		minLevel = LevelInfo
	}
	return &Logger{
		verbose:  verbose,
		w:        w,
		minLevel: minLevel,
		useUTC:   true, // osqueryd defaults to UTC
	}
}

func (l *Logger) UpdateWithOsqueryOptions(options osquery.InternalOptionList) {
	if options == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.verbose {
		if opt, ok := options["logger_min_status"]; ok && opt != nil {
			l.minLevel = parseLogLevelOption(*opt)
		}
	}
	if opt, ok := options["log_utc_time"]; ok && opt != nil {
		l.useUTC = parseUTCOption(*opt)
	} else if opt, ok := options["utc"]; ok && opt != nil {
		l.useUTC = parseUTCOption(*opt)
	}
}

// formatLog formats a log entry in glog format
// Level (I/W/E) + MMDD + HH:MM:SS.microseconds + thread_id + file:line] message
func (l *Logger) formatLog(level Level, msg string) string {
	now := time.Now()
	if l.useUTC {
		now = now.UTC()
	}

	// Get caller information (skip 3 frames: formatLog, log method, caller)
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		file = "???"
		line = 0
	}
	file = filepath.Base(file)

	// Determine level character
	levelChar := 'I'
	switch level {
	case LevelWarning:
		levelChar = 'W'
	case LevelError:
		levelChar = 'E'
	}

	// Format: I0314 15:24:36.123456 12345 file.go:123] message
	return fmt.Sprintf("%c%02d%02d %02d:%02d:%02d.%06d %d %s:%d] %s\n",
		levelChar,
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		now.Second(),
		now.Nanosecond()/1000, // Convert nanoseconds to microseconds
		pid,
		file,
		line,
		msg,
	)
}

// Info logs an informational message (only if verbose is enabled)
func (l *Logger) Info(msg string) {
	l.log(LevelInfo, msg)
}

// Infof logs a formatted informational message (only if verbose is enabled)
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LevelInfo, fmt.Sprintf(format, args...))
}

// Warning logs a warning message
func (l *Logger) Warning(msg string) {
	l.log(LevelWarning, msg)
}

// Warningf logs a formatted warning message
func (l *Logger) Warningf(format string, args ...interface{}) {
	l.log(LevelWarning, fmt.Sprintf(format, args...))
}

// Error logs an error message
func (l *Logger) Error(msg string) {
	l.log(LevelError, msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(LevelError, fmt.Sprintf(format, args...))
}

// Fatal logs an error message and exits
func (l *Logger) Fatal(msg string) {
	l.log(LevelError, msg)
	osExit(1)
}

// Fatalf logs a formatted error message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.log(LevelError, fmt.Sprintf(format, args...))
	osExit(1)
}

// log writes a log message
func (l *Logger) log(level Level, msg string) {
	if level < l.minLevel {
		return
	}

	formatted := l.formatLog(level, msg)

	l.mu.Lock()
	defer l.mu.Unlock()
	_, _ = l.w.Write([]byte(formatted))
}
