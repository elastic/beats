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

// Logger returns l writing its messages, and its string and error fields,
// without the secret values. Plugins log what they run and what their
// commands print, which can contain the values, before any event is made.
// It returns l itself for a nil Scrubber or l.
func (s *Scrubber) Logger(l *logp.Logger) *logp.Logger {
	if s == nil || l == nil {
		return l
	}
	return l.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return &scrubbingCore{Core: c, s: s}
	}))
}

type scrubbingCore struct {
	zapcore.Core
	s *Scrubber
}

func (c *scrubbingCore) With(fields []zapcore.Field) zapcore.Core {
	return &scrubbingCore{Core: c.Core.With(c.fields(fields)), s: c.s}
}

func (c *scrubbingCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(e.Level) {
		return ce.AddCore(e, c)
	}
	return ce
}

func (c *scrubbingCore) Write(e zapcore.Entry, fields []zapcore.Field) error {
	e.Message = c.s.String(e.Message)
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
		clean := c.s.String(f.String)
		if clean == f.String {
			return f, false
		}
		f.String = clean
		return f, true
	case zapcore.ErrorType:
		err, ok := f.Interface.(error)
		if !ok || c.s.String(err.Error()) == err.Error() {
			return f, false
		}
		return zap.NamedError(f.Key, c.s.Error(err)), true
	default:
		return f, false
	}
}
