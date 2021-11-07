// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package apmlog

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.elastic.co/fastjson"
)

const (
	// EnvLogFile is the environment variable that controls where the default logger writes.
	EnvLogFile = "ELASTIC_APM_LOG_FILE"

	// EnvLogLevel is the environment variable that controls the default logger's level.
	EnvLogLevel = "ELASTIC_APM_LOG_LEVEL"

	// DefaultLevel holds the default log level, if EnvLogLevel is not specified.
	DefaultLevel Level = ErrorLevel
)

var (
	// DefaultLogger is the default Logger to use, if ELASTIC_APM_LOG_* are specified.
	DefaultLogger *LevelLogger

	fastjsonPool = &sync.Pool{
		New: func() interface{} {
			return &fastjson.Writer{}
		},
	}
)

func init() {
	InitDefaultLogger()
}

// InitDefaultLogger initialises DefaultLogger using the environment variables
// ELASTIC_APM_LOG_FILE and ELASTIC_APM_LOG_LEVEL.
func InitDefaultLogger() {
	fileStr := strings.TrimSpace(os.Getenv(EnvLogFile))
	if fileStr == "" {
		DefaultLogger = nil
		return
	}

	var logWriter io.Writer
	switch strings.ToLower(fileStr) {
	case "stdout":
		logWriter = os.Stdout
	case "stderr":
		logWriter = os.Stderr
	default:
		f, err := os.Create(fileStr)
		if err != nil {
			log.Printf("failed to create %q: %s (disabling logging)", fileStr, err)
			return
		}
		logWriter = &syncFile{File: f}
	}

	logLevel := DefaultLevel
	if levelStr := strings.TrimSpace(os.Getenv(EnvLogLevel)); levelStr != "" {
		level, err := ParseLogLevel(levelStr)
		if err != nil {
			log.Printf("invalid %s %q, falling back to %q", EnvLogLevel, levelStr, logLevel)
		} else {
			logLevel = level
		}
	}
	DefaultLogger = &LevelLogger{w: logWriter, level: logLevel}
}

// Log levels.
const (
	TraceLevel Level = iota
	DebugLevel
	InfoLevel
	WarningLevel
	ErrorLevel
	CriticalLevel
	OffLevel
)

// Level represents a log level.
type Level uint32

func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "trace"
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarningLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	case CriticalLevel:
		return "critical"
	case OffLevel:
		return "off"
	}
	return ""
}

// ParseLogLevel parses s as a log level.
func ParseLogLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "trace":
		return TraceLevel, nil
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn", "warning":
		// "warn" exists for backwards compatibility;
		// "warning" is the canonical level name.
		return WarningLevel, nil
	case "error":
		return ErrorLevel, nil
	case "critical":
		return CriticalLevel, nil
	case "off":
		return OffLevel, nil
	}
	return OffLevel, fmt.Errorf("invalid log level string %q", s)
}

// LevelLogger is a level logging implementation that will log to a file,
// stdout, or stderr. The level may be updated dynamically via SetLevel.
type LevelLogger struct {
	level Level // should be accessed with sync/atomic
	w     io.Writer
}

// Level returns the current logging level.
func (l *LevelLogger) Level() Level {
	return Level(atomic.LoadUint32((*uint32)(&l.level)))
}

// SetLevel sets level as the minimum logging level.
func (l *LevelLogger) SetLevel(level Level) {
	atomic.StoreUint32((*uint32)(&l.level), uint32(level))
}

// Debugf logs a message with log.Printf, with a DEBUG prefix.
func (l *LevelLogger) Debugf(format string, args ...interface{}) {
	l.logf(DebugLevel, format, args...)
}

// Errorf logs a message with log.Printf, with an ERROR prefix.
func (l *LevelLogger) Errorf(format string, args ...interface{}) {
	l.logf(ErrorLevel, format, args...)
}

// Warningf logs a message with log.Printf, with a WARNING prefix.
func (l *LevelLogger) Warningf(format string, args ...interface{}) {
	l.logf(WarningLevel, format, args...)
}

func (l *LevelLogger) logf(level Level, format string, args ...interface{}) {
	if level < l.Level() {
		return
	}
	jw := fastjsonPool.Get().(*fastjson.Writer)
	jw.RawString(`{"level":"`)
	jw.RawString(level.String())
	jw.RawString(`","time":"`)
	jw.Time(time.Now(), time.RFC3339)
	jw.RawString(`","message":`)
	jw.String(fmt.Sprintf(format, args...))
	jw.RawString("}\n")
	l.w.Write(jw.Bytes())
	jw.Reset()
	fastjsonPool.Put(jw)
}

type syncFile struct {
	mu sync.Mutex
	*os.File
}

// Write calls f.File.Write with f.mu held, to protect multiple Tracers
// in the same process from one another.
func (f *syncFile) Write(data []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.File.Write(data)
}
