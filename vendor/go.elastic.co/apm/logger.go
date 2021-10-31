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

package apm // import "go.elastic.co/apm"

// Logger is an interface for logging, used by the tracer
// to log tracer errors and other interesting events.
type Logger interface {
	// Debugf logs a message at debug level.
	Debugf(format string, args ...interface{})

	// Errorf logs a message at error level.
	Errorf(format string, args ...interface{})
}

// WarningLogger extends Logger with a Warningf method.
//
// TODO(axw) this will be removed in v2.0.0, and the
// Warningf method will be added directly to Logger.
type WarningLogger interface {
	Logger

	// Warningf logs a message at warning level.
	Warningf(format string, args ...interface{})
}

func makeWarningLogger(l Logger) WarningLogger {
	if wl, ok := l.(WarningLogger); ok {
		return wl
	}
	return debugWarningLogger{Logger: l}
}

type debugWarningLogger struct {
	Logger
}

func (l debugWarningLogger) Warningf(format string, args ...interface{}) {
	l.Debugf(format, args...)
}
