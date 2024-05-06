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

package logp

import (
	"fmt"

	"go.uber.org/zap/zapcore"
)

// TypeKey is the default key to define log types.
//
// Different log types can be handled by different cores, the `typedLoggerCore`
// allows for choosing a different core based on a key/value pair. TypeKey
// is the default key for using the typedLoggerCore.
//
// It should be used in conjunction with the defined types on this package.
const TypeKey = "log.type"

// DefaultType is the default log type. If `log.type` is not defined a log
// entry is considered of type `DefaultType`. Those log entries should follow
// the default logging configuration.
const DefaultType = "default"

// EventType is the type for log entries containing event data.
// Beats and Elastic-Agent use this with the `typedLoggerCore` to direct
// those log entries to a different file.
const EventType = "event"

// typedLoggerCore takes two cores and directs logs entries to one of them
// with the value of the field defined by the pair `key` and `value`
//
// If `entry[key] == value` the typedCore is used, otherwise the
// defaultCore  is used.
// WARNING: The level of both cores must always be the same!
// typedLoggerCore will only use the defaultCore level to decide
// whether to log an entry or not
type typedLoggerCore struct {
	typedCore   zapcore.Core
	defaultCore zapcore.Core
	value       string
	key         string
}

func (t *typedLoggerCore) Enabled(l zapcore.Level) bool {
	return t.defaultCore.Enabled(l)
}

func (t *typedLoggerCore) With(fields []zapcore.Field) zapcore.Core {
	newCore := typedLoggerCore{
		defaultCore: t.defaultCore.With(fields),
		typedCore:   t.typedCore.With(fields),
		key:         t.key,
		value:       t.value,
	}
	return &newCore
}

func (t *typedLoggerCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if t.defaultCore.Enabled(e.Level) {
		return ce.AddCore(e, t)
	}

	return ce
}

func (t *typedLoggerCore) Sync() error {
	defaultErr := t.defaultCore.Sync()
	typedErr := t.typedCore.Sync()

	if defaultErr != nil || typedErr != nil {
		return fmt.Errorf("error syncing logger. DefaultCore: '%w', typedCore: '%w'", defaultErr, typedErr)
	}

	return nil
}

func (t *typedLoggerCore) Write(e zapcore.Entry, fields []zapcore.Field) error {
	for _, f := range fields {
		if f.Key == t.key {
			if f.String == t.value {
				return t.typedCore.Write(e, fields)
			}
			return t.defaultCore.Write(e, fields)
		}
	}

	return t.defaultCore.Write(e, fields)
}
