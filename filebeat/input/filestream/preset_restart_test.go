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
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/go-concert/unison"
)

// scriptedWatcher reports fileCounts[i] files to the gate of its i-th scan,
// and the last count to every later scan. It emits no event.
type scriptedWatcher struct {
	mockFileWatcher

	mu         sync.Mutex
	fileCounts []int
	scans      int
	admitted   int
}

func (w *scriptedWatcher) ScanOnce(
	_ unison.Canceler,
	_ loginp.FSEventSink,
	_ *loginp.Metrics,
	_ time.Duration,
	_ time.Time,
	gate loginp.ScanGate,
) {
	w.mu.Lock()
	n := w.fileCounts[min(w.scans, len(w.fileCounts)-1)]
	w.scans++
	w.mu.Unlock()

	if gate != nil && !gate(n) {
		return
	}
	w.mu.Lock()
	w.admitted++
	w.mu.Unlock()
}

func (w *scriptedWatcher) counts() (scans, admitted int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.scans, w.admitted
}

func notSetAuto(threshold int) presetInputs {
	return presetInputs{
		selection: presetSelection{name: presetAuto, threshold: threshold},
	}
}

func TestScanLoopRequestsHighCardinalityRestart(t *testing.T) {
	t.Run("stops at the first scan at the threshold", func(t *testing.T) {
		w := &scriptedWatcher{fileCounts: []int{1, 2, 3, 4}}
		p := fileProspector{
			filewatcher:  w,
			presetInputs: notSetAuto(3),
			preset: newSharedPreset(resolvedPreset{
				checkInterval: time.Millisecond,
			}),
		}

		done := make(chan struct{})
		go func() {
			defer close(done)
			p.scanLoop(t.Context(), nil, time.Time{}, func(loginp.FSEvent) {})
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			require.FailNow(
				t,
				"scanLoop must return once a restart is requested",
			)
		}

		scans, admitted := w.counts()
		assert.Equal(t, 3, scans, "scanning must stop at the triggering scan")
		assert.Equal(
			t,
			2,
			admitted,
			"the triggering scan must be dropped before it emits events",
		)
		assert.True(t, p.RestartRequested(), "a restart must be requested")
		assert.Equal(
			t,
			3,
			p.restartFileCount,
			"the file count of the triggering scan must be kept",
		)
	})

	cases := []struct {
		name   string
		inputs presetInputs
		preset resolvedPreset
	}{
		{
			name:   "already high_cardinality",
			inputs: notSetAuto(3),
			preset: resolvedPreset{highCardinality: true},
		},
		{
			name: "explicit default",
			inputs: presetInputs{selection: presetSelection{
				name:      presetDefault,
				explicit:  true,
				threshold: 3,
			}},
		},
		{
			name: "copytruncate",
			inputs: presetInputs{
				selection:    presetSelection{name: presetAuto, threshold: 3},
				forceDefault: true,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name+" never restarts", func(t *testing.T) {
			w := &scriptedWatcher{fileCounts: []int{10}}
			tc.preset.checkInterval = time.Millisecond
			p := fileProspector{
				filewatcher:  w,
				presetInputs: tc.inputs,
				preset:       newSharedPreset(tc.preset),
			}

			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan struct{})
			go func() {
				defer close(done)
				p.scanLoop(ctx, nil, time.Time{}, func(loginp.FSEvent) {})
			}()
			assert.Eventually(
				t,
				func() bool {
					_, admitted := w.counts()
					return admitted >= 5
				},
				5*time.Second,
				time.Millisecond,
				"every scan must be admitted",
			)
			cancel()
			<-done

			assert.False(
				t,
				p.RestartRequested(),
				"no restart must be requested",
			)
		})
	}
}

func TestProspectorRunAndRestartIntoHighCardinality(t *testing.T) {
	logger, observed := logptest.NewTestingLoggerWithObserver(t, "")
	w := &scriptedWatcher{fileCounts: []int{2, 3}}
	p := &fileProspector{
		logger:       logger,
		identifier:   mustPathIdentifier(false),
		filewatcher:  w,
		presetInputs: notSetAuto(3),
		preset: newSharedPreset(resolvedPreset{
			checkInterval: time.Millisecond,
		}),
	}
	p.presetInputs.closeOnEOF = true
	p.presetInputs.checkInterval = time.Millisecond
	p.presetInputs.checkIntervalSet = true

	hg := newTestHarvesterGroup()
	ctx := input.Context{Logger: logger, Cancelation: t.Context()}
	p.Run(ctx, nil, hg, nil)

	require.True(t, p.RestartRequested(), "Run must return to restart")
	assert.NotContains(
		t,
		hg.events,
		harvesterEvent(harvesterGroupStop{}),
		"the input stops the harvesters on a restart, not Run",
	)

	p.Restart()
	assert.False(t, p.RestartRequested(), "Restart must clear the request")
	assert.True(
		t,
		p.preset.load().highCardinality,
		"Restart must resolve high_cardinality",
	)
	assert.Equal(
		t,
		uint64(highCardinalityHarvesterLimit),
		p.HarvesterLimit(),
		"the harvester limit must be the high_cardinality one",
	)
	transitions := observed.FilterMessageSnippet(
		"preset changed from default to high_cardinality",
	)
	assert.Equal(
		t,
		1,
		transitions.Len(),
		"the transition must be logged once",
	)
	onEOFWarnings := observed.FilterMessageSnippet(
		"'close.reader.on_eof' is ignored",
	)
	assert.Equal(
		t,
		1,
		onEOFWarnings.Len(),
		"the ignored close.reader.on_eof must be reported on the restart",
	)

	runCtx, cancel := context.WithCancel(t.Context())
	hg = newTestHarvesterGroup()
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Run(input.Context{Logger: logger, Cancelation: runCtx}, nil, hg, nil)
	}()
	assert.Eventually(
		t,
		func() bool {
			scans, _ := w.counts()
			return scans >= 5
		},
		5*time.Second,
		time.Millisecond,
		"the restarted Run must keep scanning",
	)
	cancel()
	<-done

	assert.False(t, p.RestartRequested(), "high_cardinality must not restart")
	assert.Equal(
		t,
		[]harvesterEvent{harvesterGroupStop{}},
		hg.events,
		"a Run that is not restarting must stop its harvesters",
	)
}

func TestCopyTruncateProspectorNeverRestarts(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	w := &scriptedWatcher{fileCounts: []int{10}}
	fp := fileProspector{
		logger:      logger,
		identifier:  mustPathIdentifier(false),
		filewatcher: w,
		presetInputs: presetInputs{
			selection:    presetSelection{name: presetAuto, threshold: 3},
			forceDefault: true,
		},
		preset: newSharedPreset(resolvedPreset{checkInterval: time.Millisecond}),
	}
	p := &copyTruncateFileProspector{fileProspector: fp}

	ctx, cancel := context.WithCancel(t.Context())
	hg := newTestHarvesterGroup()
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Run(input.Context{Logger: logger, Cancelation: ctx}, nil, hg, nil)
	}()
	assert.Eventually(
		t,
		func() bool {
			_, admitted := w.counts()
			return admitted >= 5
		},
		5*time.Second,
		time.Millisecond,
		"every scan must be admitted",
	)
	cancel()
	<-done

	assert.False(t, p.RestartRequested(), "copytruncate must never restart")
	assert.Equal(
		t,
		[]harvesterEvent{harvesterGroupStop{}},
		hg.events,
		"copytruncate must stop its harvesters when Run returns",
	)
}

func TestFileWatcherScanGateAndReset(t *testing.T) {
	dir := t.TempDir()
	write := func(name string) string {
		path := filepath.Join(dir, name)
		require.NoError(
			t,
			os.WriteFile(path, []byte(name+" line\n"), 0o644),
			"cannot write %s",
			path,
		)
		return path
	}
	a, b := write("a.log"), write("b.log")
	fw := createWatcherWithConfig(
		t,
		logptest.NewTestingLogger(t, ""),
		[]string{filepath.Join(dir, "*.log")},
		"scanner:\n  fingerprint.enabled: false\n",
	)

	type event struct {
		path string
		op   loginp.Operation
	}
	scan := func(gate loginp.ScanGate) []event {
		var got []event
		fw.ScanOnce(t.Context(), func(e loginp.FSEvent) {
			got = append(got, event{path: e.NewPath, op: e.Op})
		}, newTestMetrics(), 0, time.Time{}, gate)
		slices.SortFunc(got, func(x, y event) int {
			return strings.Compare(x.path, y.path)
		})
		return got
	}
	created := func(paths ...string) []event {
		evts := make([]event, 0, len(paths))
		for _, p := range paths {
			evts = append(evts, event{path: p, op: loginp.OpCreate})
		}
		return evts
	}

	assert.Equal(
		t,
		created(a, b),
		scan(nil),
		"the first scan must create both files",
	)
	assert.Empty(t, scan(nil), "an unchanged scan must emit nothing")

	c := write("c.log")
	var gateCount int
	dropped := scan(func(n int) bool {
		gateCount = n
		return false
	})
	assert.Empty(t, dropped, "a dropped scan must emit nothing")
	assert.Equal(t, 3, gateCount, "the gate must see every file of the scan")
	assert.Equal(
		t,
		created(c),
		scan(func(int) bool { return true }),
		"a dropped scan must leave the previous scan state untouched",
	)

	fw.HarvesterClosed(loginp.HarvesterStatus{ID: "some-id", Size: 1})
	fw.reset()
	assert.Equal(
		t,
		created(a, b, c),
		scan(nil),
		"after a reset every file must be reported as created",
	)
	assert.False(
		t,
		fw.hasClosedHarvesters(),
		"a reset must drop the pending harvester close notifications",
	)
}
