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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerWithOptions(t *testing.T) {
	core1, observed1 := observer.New(zapcore.DebugLevel)
	core2, observed2 := observer.New(zapcore.DebugLevel)

	logger1 := NewLogger("bo", zap.WrapCore(func(in zapcore.Core) zapcore.Core {
		return zapcore.NewTee(in, core1)
	}))
	logger2 := logger1.WithOptions(zap.WrapCore(func(in zapcore.Core) zapcore.Core {
		return zapcore.NewTee(in, core2)
	}))

	logger1.Info("hello logger1")             // should just go to the first observer
	logger2.Info("hello logger1 and logger2") // should go to both observers

	observedEntries1 := observed1.All()
	require.Len(t, observedEntries1, 2)
	assert.Equal(t, "hello logger1", observedEntries1[0].Message)
	assert.Equal(t, "hello logger1 and logger2", observedEntries1[1].Message)

	observedEntries2 := observed2.All()
	require.Len(t, observedEntries2, 1)
	assert.Equal(t, "hello logger1 and logger2", observedEntries2[0].Message)
}

func TestNewInMemory(t *testing.T) {
	log, buff := NewInMemory("in_memory", ConsoleEncoderConfig())

	log.Debugw("a debug message", "debug_key", "debug_val")
	log.Infow("a info message", "info_key", "info_val")
	log.Warnw("a warn message", "warn_key", "warn_val")
	log.Errorw("an error message", "error_key", "error_val")

	logs := strings.Split(strings.TrimSpace(buff.String()), "\n")
	assert.Len(t, logs, 4, "expected 4 log entries")

	assert.Contains(t, logs[0], "a debug message")
	assert.Contains(t, logs[0], "debug_key")
	assert.Contains(t, logs[0], "debug_val")

	assert.Contains(t, logs[1], "a info message")
	assert.Contains(t, logs[1], "info_key")
	assert.Contains(t, logs[1], "info_val")

	assert.Contains(t, logs[2], "a warn message")
	assert.Contains(t, logs[2], "warn_key")
	assert.Contains(t, logs[2], "warn_val")

	assert.Contains(t, logs[3], "an error message")
	assert.Contains(t, logs[3], "error_key")
	assert.Contains(t, logs[3], "error_val")
}

func TestLoggerWithLazy(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	root, err := NewZapLogger(zap.New(core))
	require.NoError(t, err, "constructing zap logger")

	child := root.WithLazy(
		zap.String("operation", "create"),
		zap.String("source_file", "src-1"),
	)
	child.Infof("hello %s", "world")

	entries := observed.All()
	require.Len(t, entries, 1, "expected one emitted entry")
	fields := map[string]string{}
	for _, f := range entries[0].Context {
		if f.Type == zapcore.StringType {
			fields[f.Key] = f.String
		}
	}
	assert.Equal(t, "create", fields["operation"], "operation field")
	assert.Equal(t, "src-1", fields["source_file"], "source_file field")
	assert.Equal(t, "hello world", entries[0].Message, "formatted message")
}

func TestLoggerWithLazyBelowLevelEmitsNothing(t *testing.T) {
	// A Debugf call against an Info-level logger should produce no observer
	// entries, even when the logger was built via WithLazy.
	core, observed := observer.New(zapcore.InfoLevel)
	root, err := NewZapLogger(zap.New(core))
	require.NoError(t, err, "constructing zap logger")

	child := root.WithLazy(
		zap.String("operation", "create"),
		zap.String("source_file", "src-1"),
	)
	child.Debugf("debugf below configured level")

	assert.Empty(t, observed.All(), "no entries should be emitted below the configured level")
}

func TestLoggerHasSelector(t *testing.T) {
	logger := newLogger(zap.NewNop(), map[string]struct{}{
		"config": {},
		"*":      {},
	})

	assert.True(t, logger.HasSelector("config"))
	assert.False(t, logger.HasSelector("publish"))
}
