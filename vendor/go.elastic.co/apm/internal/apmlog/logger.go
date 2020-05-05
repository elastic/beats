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
	"time"

	"go.elastic.co/fastjson"
)

var (
	// DefaultLogger is the default Logger to use, if ELASTIC_APM_LOG_* are specified.
	DefaultLogger Logger

	fastjsonPool = &sync.Pool{
		New: func() interface{} {
			return &fastjson.Writer{}
		},
	}
)

func init() {
	initDefaultLogger()
}

func initDefaultLogger() {
	fileStr := strings.TrimSpace(os.Getenv("ELASTIC_APM_LOG_FILE"))
	if fileStr == "" {
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

	logLevel := errorLevel
	if levelStr := strings.TrimSpace(os.Getenv("ELASTIC_APM_LOG_LEVEL")); levelStr != "" {
		level, err := parseLogLevel(levelStr)
		if err != nil {
			log.Printf("invalid ELASTIC_APM_LOG_LEVEL %q, falling back to %q", levelStr, logLevel)
		} else {
			logLevel = level
		}
	}
	DefaultLogger = levelLogger{w: logWriter, level: logLevel}
}

const (
	debugLevel logLevel = iota
	infoLevel
	warnLevel
	errorLevel
	noLevel
)

type logLevel uint8

func (l logLevel) String() string {
	switch l {
	case debugLevel:
		return "debug"
	case infoLevel:
		return "info"
	case warnLevel:
		return "warn"
	case errorLevel:
		return "error"
	}
	return ""
}

func parseLogLevel(s string) (logLevel, error) {
	switch strings.ToLower(s) {
	case "debug":
		return debugLevel, nil
	case "info":
		return infoLevel, nil
	case "warn":
		return warnLevel, nil
	case "error":
		return errorLevel, nil
	}
	return noLevel, fmt.Errorf("invalid log level string %q", s)
}

// Logger provides methods for logging.
type Logger interface {
	Debugf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Warningf(format string, args ...interface{})
}

type levelLogger struct {
	w     io.Writer
	level logLevel
}

// Debugf logs a message with log.Printf, with a DEBUG prefix.
func (l levelLogger) Debugf(format string, args ...interface{}) {
	l.logf(debugLevel, format, args...)
}

// Errorf logs a message with log.Printf, with an ERROR prefix.
func (l levelLogger) Errorf(format string, args ...interface{}) {
	l.logf(errorLevel, format, args...)
}

// Warningf logs a message with log.Printf, with a WARNING prefix.
func (l levelLogger) Warningf(format string, args ...interface{}) {
	l.logf(warnLevel, format, args...)
}

func (l levelLogger) logf(level logLevel, format string, args ...interface{}) {
	if level < l.level {
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
