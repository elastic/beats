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

package secretstores

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Logger returns logger writing its messages, and its string and error
// fields, without the secret values. Plugins log what they run and what their
// commands print, which can contain the values, before any event is made.
// Other field types, such as Stringer or Any, are written as they are.
// It returns logger itself for a nil Redactor or logger.
func (r *Redactor) Logger(logger *logp.Logger) *logp.Logger {
	if r == nil || logger == nil {
		return logger
	}
	return logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return &scrubbingCore{Core: c, scrub: r.scrub}
	}))
}

type scrubbingCore struct {
	zapcore.Core
	scrub func(string) string
}

func (c *scrubbingCore) With(fields []zapcore.Field) zapcore.Core {
	return &scrubbingCore{Core: c.Core.With(c.fields(fields)), scrub: c.scrub}
}

func (c *scrubbingCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// The wrapped core decides: logp filters debug entries by selector in
	// Check, not only by level.
	if c.Core.Check(e, nil) == nil {
		return ce
	}
	return ce.AddCore(e, c)
}

func (c *scrubbingCore) Write(e zapcore.Entry, fields []zapcore.Field) error {
	e.Message = c.scrub(e.Message)
	return c.Core.Write(e, c.fields(fields))
}

// fields returns fields without the secret values, copying them only if one
// changes.
func (c *scrubbingCore) fields(fields []zapcore.Field) []zapcore.Field {
	var out []zapcore.Field
	for i, f := range fields {
		scrubbed, changed := c.field(f)
		if !changed {
			continue
		}
		if out == nil {
			out = make([]zapcore.Field, len(fields))
			copy(out, fields)
		}
		out[i] = scrubbed
	}
	if out == nil {
		return fields
	}
	return out
}

func (c *scrubbingCore) field(f zapcore.Field) (zapcore.Field, bool) {
	switch f.Type {
	case zapcore.StringType:
		clean := c.scrub(f.String)
		if clean == f.String {
			return f, false
		}
		f.String = clean
		return f, true
	case zapcore.ErrorType:
		err, ok := f.Interface.(error)
		if !ok {
			return f, false
		}
		msg := err.Error()
		clean := c.scrub(msg)
		if clean == msg {
			return f, false
		}
		// Encoded as the error message would be.
		return zap.String(f.Key, clean), true
	default:
		return f, false
	}
}
