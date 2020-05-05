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

package apmtest

import "fmt"

// RecordLogger is an implementation of apm.Logger, recording log entries.
type RecordLogger struct {
	Records []LogRecord
}

// Debugf logs debug messages.
func (l *RecordLogger) Debugf(format string, args ...interface{}) {
	l.logf("debug", format, args...)
}

// Errorf logs error messages.
func (l *RecordLogger) Errorf(format string, args ...interface{}) {
	l.logf("error", format, args...)
}

// Warningf logs error messages.
func (l *RecordLogger) Warningf(format string, args ...interface{}) {
	l.logf("warning", format, args...)
}

func (l *RecordLogger) logf(level string, format string, args ...interface{}) {
	l.Records = append(l.Records, LogRecord{
		Level:   level,
		Format:  format,
		Message: fmt.Sprintf(format, args...),
	})
}

// LogRecord holds the details of a log record.
type LogRecord struct {
	// Level is the log level: "debug", "error", or "warning".
	Level string

	// Format is the log message format, like "Thingy did foo %d times".
	Format string

	// Message is the formatted message.
	Message string
}
