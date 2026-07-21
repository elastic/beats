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

package filestream

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestLoggerWithEvent(t *testing.T) {
	minimalEvent := loginp.FSEvent{Op: loginp.OpCreate, SrcID: "src-1"}
	fullEvent := loginp.FSEvent{
		Op:         loginp.OpRename,
		SrcID:      "src-2",
		NewPath:    "/var/log/new.log",
		OldPath:    "/var/log/old.log",
		Descriptor: loginp.FileDescriptor{Fingerprint: loginp.FingerprintID{Sum: "abc123"}},
	}

	cases := []struct {
		name        string
		event       loginp.FSEvent
		loggerLevel zapcore.Level
		emit        func(*logp.Logger)
		wantEntries int
		wantFields  map[string]string
		wantAbsent  []string
	}{
		{
			name:        "warnf with minimal event enriches and emits",
			event:       minimalEvent,
			loggerLevel: zapcore.InfoLevel,
			emit:        func(l *logp.Logger) { l.Warnf("hello") },
			wantEntries: 1,
			wantFields:  map[string]string{"operation": "create"},
			wantAbsent:  []string{"source_file", "fingerprint", "os_id", "new_path", "old_path"},
		},
		{
			name:        "warnf with all event fields populates every key",
			event:       fullEvent,
			loggerLevel: zapcore.InfoLevel,
			emit:        func(l *logp.Logger) { l.Warnf("hello") },
			wantEntries: 1,
			wantFields: map[string]string{
				"operation": "rename",
				"new_path":  "/var/log/new.log",
				"old_path":  "/var/log/old.log",
			},
			wantAbsent: []string{"source_file", "fingerprint", "os_id"},
		},
		{
			name:        "debugf is a no-op when debug logging is disabled",
			event:       minimalEvent,
			loggerLevel: zapcore.InfoLevel,
			emit:        func(l *logp.Logger) { l.Debugf("noisy %d", 1) },
			wantEntries: 0,
		},
		{
			name:        "debugf enriches and emits when debug logging is enabled",
			event:       minimalEvent,
			loggerLevel: zapcore.DebugLevel,
			emit:        func(l *logp.Logger) { l.Debugf("noisy %d", 1) },
			wantEntries: 1,
			wantFields:  map[string]string{"operation": "create"},
			wantAbsent:  []string{"source_file", "fingerprint"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			core, observed := observer.New(tc.loggerLevel)
			base, err := logp.NewZapLogger(zap.New(core))
			require.NoError(t, err, "constructing zap logger")

			log := loggerWithEvent(base, tc.event)
			tc.emit(log)

			entries := observed.All()
			require.Lenf(t, entries, tc.wantEntries, "unexpected number of emitted entries")

			for i, e := range entries {
				fields := map[string]string{}
				for _, f := range e.Context {
					if f.Type == zapcore.StringType {
						fields[f.Key] = f.String
					}
				}
				for k, v := range tc.wantFields {
					assert.Equalf(t, v, fields[k], "entry %d: field %q value mismatch", i, k)
				}
				for _, k := range tc.wantAbsent {
					_, ok := fields[k]
					assert.Falsef(t, ok, "entry %d: field %q must not be present", i, k)
				}
			}
		})
	}
}

// TestLoggerWithEventAllocs guards the per-FS-event enrichment allocation
// count: the fields slice plus the WithLazy logger clone.
func TestLoggerWithEventAllocs(t *testing.T) {
	logger := logp.NewNopLogger()
	event := loginp.FSEvent{
		Op:      loginp.OpRename,
		SrcID:   "filestream::my-input-id::fingerprint::2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a",
		NewPath: "/var/log/new.log",
		OldPath: "/var/log/old.log",
		Descriptor: loginp.FileDescriptor{
			Fingerprint: loginp.FingerprintID{
				Sum: "2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a",
			},
		},
	}

	allocs := testing.AllocsPerRun(1000, func() {
		loggerWithEvent(logger, event)
	})
	assert.LessOrEqual(t, allocs, 8.0, "loggerWithEvent allocated more than expected")
}

func BenchmarkLoggerWithEvent(b *testing.B) {
	event := loginp.FSEvent{
		Op:      loginp.OpWrite,
		SrcID:   "filestream::my-input-id::fingerprint::2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a",
		NewPath: "/var/log/app/application.log",
		Descriptor: loginp.FileDescriptor{
			Fingerprint: loginp.FingerprintID{
				Sum: "2edc986847e209b4016e141a6dc8716d3207350f416969382d431539bf292e4a",
			},
		},
	}

	b.Run("debug_suppressed", func(b *testing.B) {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zapcore.InfoLevel,
		)
		logger, err := logp.NewZapLogger(zap.New(core))
		require.NoError(b, err, "constructing zap logger")

		b.ReportAllocs()
		for b.Loop() {
			log := loggerWithEvent(logger, event)
			log.Debugf("File %s has been updated", event.NewPath)
		}
	})

	b.Run("emitted", func(b *testing.B) {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zapcore.DebugLevel,
		)
		logger, err := logp.NewZapLogger(zap.New(core))
		require.NoError(b, err, "constructing zap logger")

		b.ReportAllocs()
		for b.Loop() {
			log := loggerWithEvent(logger, event)
			log.Debugf("File %s has been updated", event.NewPath)
		}
	})
}
