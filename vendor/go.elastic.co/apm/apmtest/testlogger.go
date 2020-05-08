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

// TestLogger is an implementation of apm.Logger,
// logging to a testing.T.
type TestLogger struct {
	l LogfLogger
}

// NewTestLogger returns a new TestLogger that logs messages to l.
func NewTestLogger(l LogfLogger) TestLogger {
	return TestLogger{l: l}
}

// Debugf logs debug messages.
func (t TestLogger) Debugf(format string, args ...interface{}) {
	t.l.Logf("[DEBUG] "+format, args...)
}

// Errorf logs error messages.
func (t TestLogger) Errorf(format string, args ...interface{}) {
	t.l.Logf("[ERROR] "+format, args...)
}

// LogfLogger is an interface with the a Logf method,
// implemented by *testing.T and *testing.B.
type LogfLogger interface {
	Logf(string, ...interface{})
}
