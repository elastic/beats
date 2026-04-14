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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestLazyLog(t *testing.T) {
	minimalEvent := loginp.FSEvent{Op: loginp.OpCreate, SrcID: "src-1"}
	fullEvent := loginp.FSEvent{
		Op:         loginp.OpRename,
		SrcID:      "src-2",
		NewPath:    "/var/log/new.log",
		OldPath:    "/var/log/old.log",
		Descriptor: loginp.FileDescriptor{Fingerprint: "abc123"},
	}

	cases := []struct {
		name         string
		event        loginp.FSEvent
		loggerLevel  zapcore.Level
		calls        []func(*lazyLog)
		wantEntries  int
		wantFields   map[string]string // expected on each emitted entry
		wantAbsent   []string          // must not appear on any emitted entry
		wantEnriched bool
	}{
		{
			name:         "warnf with minimal event enriches and emits",
			event:        minimalEvent,
			loggerLevel:  zapcore.InfoLevel,
			calls:        []func(*lazyLog){func(l *lazyLog) { l.Warnf("hello") }},
			wantEntries:  1,
			wantFields:   map[string]string{"operation": "create", "source_file": "src-1"},
			wantAbsent:   []string{"fingerprint", "os_id", "new_path", "old_path"},
			wantEnriched: true,
		},
		{
			name:        "warnf with all event fields populates every key",
			event:       fullEvent,
			loggerLevel: zapcore.InfoLevel,
			calls:       []func(*lazyLog){func(l *lazyLog) { l.Warnf("hello") }},
			wantEntries: 1,
			wantFields: map[string]string{
				"operation":   "rename",
				"source_file": "src-2",
				"fingerprint": "abc123",
				"new_path":    "/var/log/new.log",
				"old_path":    "/var/log/old.log",
			},
			wantAbsent:   []string{"os_id"},
			wantEnriched: true,
		},
		{
			name:         "errorf enriches and emits",
			event:        minimalEvent,
			loggerLevel:  zapcore.InfoLevel,
			calls:        []func(*lazyLog){func(l *lazyLog) { l.Errorf("oops %d", 1) }},
			wantEntries:  1,
			wantFields:   map[string]string{"operation": "create", "source_file": "src-1"},
			wantEnriched: true,
		},
		{
			name:         "debugf is a no-op when debug logging is disabled",
			event:        minimalEvent,
			loggerLevel:  zapcore.InfoLevel,
			calls:        []func(*lazyLog){func(l *lazyLog) { l.Debugf("noisy %d", 1) }},
			wantEntries:  0,
			wantEnriched: false,
		},
		{
			name:         "debugf enriches and emits when debug logging is enabled",
			event:        minimalEvent,
			loggerLevel:  zapcore.DebugLevel,
			calls:        []func(*lazyLog){func(l *lazyLog) { l.Debugf("noisy %d", 1) }},
			wantEntries:  1,
			wantFields:   map[string]string{"operation": "create", "source_file": "src-1"},
			wantEnriched: true,
		},
		{
			name:        "two warnf calls share a single enriched logger",
			event:       minimalEvent,
			loggerLevel: zapcore.InfoLevel,
			calls: []func(*lazyLog){
				func(l *lazyLog) { l.Warnf("first") },
				func(l *lazyLog) { l.Warnf("second") },
			},
			wantEntries:  2,
			wantFields:   map[string]string{"operation": "create", "source_file": "src-1"},
			wantEnriched: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			core, observed := observer.New(tc.loggerLevel)
			log, err := logp.NewZapLogger(zap.New(core))
			require.NoError(t, err, "constructing zap logger")

			ll := loggerWithEvent(log, tc.event)
			for _, call := range tc.calls {
				call(&ll)
			}

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

			assert.Equalf(t, tc.wantEnriched, ll.enriched, "enriched flag mismatch")

			if tc.wantEnriched {
				before := ll.log
				assert.Samef(t, before, ll.enrich(), "enrich must memoize the enriched logger")
			}
		})
	}
}
