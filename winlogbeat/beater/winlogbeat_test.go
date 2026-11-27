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

package beater

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/paths"
)

func TestLegacyStateReadInRun(t *testing.T) {
	data := t.TempDir()
	file := filepath.Join(data, ".winlogbeat.yml")
	require.NoError(t, os.WriteFile(file, []byte("event_logs: ["), 0o600), "failed to write legacy state file")

	cfg, err := conf.NewConfigWithYAML([]byte("event_logs:\n- name: Security\n"), "test")
	require.NoError(t, err, "failed to parse test config")
	b := &beat.Beat{
		Info: beat.Info{
			Logger: logptest.NewTestingLogger(t, ""),
			Paths:  &paths.Path{Home: data, Config: data, Data: data, Logs: data},
		},
		Config:     &beat.BeatConfig{},
		BeatConfig: cfg,
		// Skips eventlog.New, which only works on Windows.
		InSetupCmd: true,
	}
	bt, err := New(b, nil)
	require.NoError(t, err, "New should not read the legacy state file")
	w, ok := bt.(*winlogbeat)
	require.True(t, ok, "New returned %T; want *winlogbeat", bt)

	err = w.loadLegacyState()
	require.Error(t, err, "corrupt legacy state file should fail the load in Run")
	assert.Contains(t, err.Error(), "failed to read persisted state from "+file, "unexpected error text")

	require.NoError(t, os.WriteFile(file, []byte("event_logs:\n- name: Security\n  record_number: 42\n"), 0o600), "failed to write legacy state file")
	require.NoError(t, w.loadLegacyState(), "valid legacy state file should load")
	st, ok := w.initialState("Security")
	assert.True(t, ok, "initialState should find the loaded event log")
	assert.Equal(t, uint64(42), st.RecordNumber, "initialState should return the loaded record number")
}

func TestNewNoFilebeatModuleLogs(t *testing.T) {
	// The home directory has no module directory and the output is not
	// Elasticsearch.
	home := t.TempDir()
	cfg, err := conf.NewConfigWithYAML([]byte("event_logs:\n- name: Security\n"), "test")
	require.NoError(t, err, "failed to parse test config")
	logger, logs := logptest.NewTestingLoggerWithObserver(t, "")
	b := &beat.Beat{
		Info: beat.Info{
			Logger: logger,
			Paths:  &paths.Path{Home: home, Config: home, Data: home, Logs: home},
		},
		Config:     &beat.BeatConfig{},
		BeatConfig: cfg,
		// Skips eventlog.New, which only works on Windows.
		InSetupCmd: true,
	}
	_, err = New(b, nil)
	require.NoError(t, err, "New failed")

	assert.Zero(t, logs.FilterMessageSnippet("Not loading modules").Len(), "Filebeat module directory error logged")
	assert.Zero(t, logs.FilterMessageSnippet("unable to load the ingest pipelines for the configured modules").Len(), "Filebeat pipelines warning logged")
	assert.NotNil(t, b.OverwritePipelinesCallback, "Winlogbeat pipeline callback not installed")
}

func TestStopTracker(t *testing.T) {
	t.Run("all distinct names stop", func(t *testing.T) {
		var calls atomic.Int32
		tr := newStopTracker(2, func() { calls.Add(1) })
		tr.stopped("a")
		assert.Equal(t, int32(0), calls.Load(), "stop called with one log still running")
		tr.stopped("b")
		assert.Equal(t, int32(1), calls.Load(), "stop should be called once all logs stopped")
	})

	t.Run("repeated name counts once", func(t *testing.T) {
		var calls atomic.Int32
		tr := newStopTracker(2, func() { calls.Add(1) })
		tr.stopped("a")
		tr.stopped("a")
		assert.Equal(t, int32(0), calls.Load(), "stop called with one log still running")
	})

	t.Run("stop called exactly once", func(t *testing.T) {
		var calls atomic.Int32
		tr := newStopTracker(1, func() { calls.Add(1) })
		tr.stopped("a")
		tr.stopped("a")
		tr.stopped("a")
		assert.Equal(t, int32(1), calls.Load(), "stop should be called exactly once")
	})

	t.Run("concurrent", func(t *testing.T) {
		var calls atomic.Int32
		tr := newStopTracker(3, func() { calls.Add(1) })
		var wg sync.WaitGroup
		for i := range 3 {
			wg.Go(func() {
				tr.stopped(fmt.Sprint(i))
			})
		}
		wg.Wait()
		assert.Equal(t, int32(1), calls.Load(), "stop should be called exactly once")
	})
}
