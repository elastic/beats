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

//go:build !windows && !nacl && !plan9
// +build !windows,!nacl,!plan9

package logp

import (
	"log/syslog"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type syslogCore struct {
	zapcore.LevelEnabler
	encoder zapcore.Encoder
	writer  *syslog.Writer
	fields  []zapcore.Field
}

// newSyslog returns a new Core that outputs to syslog.
func newSyslog(encoder zapcore.Encoder, enab zapcore.LevelEnabler) (zapcore.Core, error) {
	// Initialize a syslog writer.
	writer, err := syslog.New(syslog.LOG_ERR|syslog.LOG_LOCAL0, filepath.Base(os.Args[0]))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get a syslog writer")
	}

	return &syslogCore{
		LevelEnabler: enab,
		encoder:      encoder,
		writer:       writer,
	}, nil
}

func (c *syslogCore) With(fields []zapcore.Field) zapcore.Core {
	clone := c.Clone()
	clone.fields = append(clone.fields, fields...)
	return clone
}

func (c *syslogCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *syslogCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	buffer, err := c.encoder.EncodeEntry(entry, fields)
	if err != nil {
		return errors.Wrap(err, "failed to encode entry")
	}

	// Console encoder writes tabs which don't render nicely with syslog.
	replaceTabsWithSpaces(buffer.Bytes(), 4)

	msg := buffer.String()
	switch entry.Level {
	case zapcore.DebugLevel:
		return c.writer.Debug(msg)
	case zapcore.InfoLevel:
		return c.writer.Info(msg)
	case zapcore.WarnLevel:
		return c.writer.Warning(msg)
	case zapcore.ErrorLevel:
		return c.writer.Err(msg)
	case zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return c.writer.Crit(msg)
	default:
		return errors.Errorf("unhandled log level: %v", entry.Level)
	}
}

func (c *syslogCore) Sync() error {
	return nil
}

func (c *syslogCore) Clone() *syslogCore {
	clone := *c
	clone.encoder = c.encoder.Clone()
	clone.fields = make([]zapcore.Field, len(c.fields), len(c.fields)+10)
	copy(clone.fields, c.fields)
	return &clone
}

func replaceTabsWithSpaces(b []byte, n int) {
	var count = 0
	for i, v := range b {
		if v == '\t' {
			b[i] = ' '

			count++
			if n >= 0 && count >= n {
				return
			}
		}
	}
}
