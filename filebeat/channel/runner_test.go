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

package channel

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/processors"
	_ "github.com/elastic/beats/v7/libbeat/processors/actions"
	"github.com/elastic/beats/v7/libbeat/processors/actions/addfields"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_cloud_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_kubernetes_metadata"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/paths"
)

func TestProcessorsForConfig(t *testing.T) {
	testCases := map[string]struct {
		beatInfo       beat.Info
		configStr      string
		clientCfg      beat.ClientConfig
		event          beat.Event
		expectedFields map[string]string
	}{
		"Simple static index": {
			configStr: "index: 'test'",
			expectedFields: map[string]string{
				"@metadata.raw_index": "test",
			},
		},
		"Index with agent info + timestamp": {
			beatInfo:  beat.Info{Beat: "TestBeat", Version: "3.9.27", Logger: logptest.NewTestingLogger(t, "")},
			configStr: "index: 'beat-%{[agent.name]}-%{[agent.version]}-%{+yyyy.MM.dd}'",
			event:     beat.Event{Timestamp: time.Date(1999, time.December, 31, 23, 0, 0, 0, time.UTC)},
			expectedFields: map[string]string{
				"@metadata.raw_index": "beat-TestBeat-3.9.27-1999.12.31",
			},
		},
		"Set index in ClientConfig": {
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(&setRawIndex{"clientCfgIndex"}),
				},
			},
			expectedFields: map[string]string{
				"@metadata.raw_index": "clientCfgIndex",
			},
		},
		"ClientConfig processor runs after beat input Index": {
			configStr: "index: 'test'",
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(&setRawIndex{"clientCfgIndex"}),
				},
			},
			expectedFields: map[string]string{
				"@metadata.raw_index": "clientCfgIndex",
			},
		},
		"Set field in input config": {
			configStr: `processors: [add_fields: {fields: {testField: inputConfig}}]`,
			expectedFields: map[string]string{
				"fields.testField": "inputConfig",
			},
		},
		"Set field in ClientConfig": {
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(addfields.NewAddFields(mapstr.M{
						"fields": mapstr.M{"testField": "clientConfig"},
					}, false, true)),
				},
			},
			expectedFields: map[string]string{
				"fields.testField": "clientConfig",
			},
		},
		"Input config processors run after ClientConfig": {
			configStr: `processors: [add_fields: {fields: {testField: inputConfig}}]`,
			clientCfg: beat.ClientConfig{
				Processing: beat.ProcessingConfig{
					Processor: makeProcessors(addfields.NewAddFields(mapstr.M{
						"fields": mapstr.M{"testField": "clientConfig"},
					}, false, true)),
				},
			},
			expectedFields: map[string]string{
				"fields.testField": "inputConfig",
			},
		},
	}
	for description, test := range testCases {
		if test.event.Fields == nil {
			test.event.Fields = mapstr.M{}
		}
		config, err := conf.NewConfigFrom(test.configStr)
		if err != nil {
			t.Errorf("[%s] %v", description, err)
			continue
		}

		editor, sharedProcs, err := newCommonConfigEditor(test.beatInfo, config)
		if err != nil {
			t.Errorf("[%s] %v", description, err)
			continue
		}
		t.Cleanup(func() { _ = sharedProcs.Close() })

		clientCfg, err := editor(test.clientCfg)
		require.NoError(t, err)

		processors := clientCfg.Processing.Processor
		processedEvent, err := processors.Run(&test.event)
		// We don't check if err != nil, because we are testing the final outcome
		// of running the processors, including when some of them fail.
		if processedEvent == nil {
			t.Errorf("[%s] Unexpected fatal error running processors: %v\n",
				description, err)
		}
		for key, value := range test.expectedFields {
			field, err := processedEvent.GetValue(key)
			if err != nil {
				t.Errorf("[%s] Couldn't get field %s from event: %v", description, key, err)
				continue
			}
			assert.Equal(t, field, value)
			fieldStr, ok := field.(string)
			if !ok {
				// Note that requiring a string here is just to simplify the test setup,
				// not a requirement of the underlying api.
				t.Errorf("[%s] Field [%s] should be a string", description, key)
				continue
			}
			if fieldStr != value {
				t.Errorf("[%s] Event field [%s]: expected [%s], got [%s]", description, key, value, fieldStr)
			}
		}
	}
}

func TestProcessorsForConfigIsFlat(t *testing.T) {
	// This test is regrettable, and exists because of inconsistencies in
	// processor handling between processors.Processors and processing.group
	// (which implements beat.ProcessorList) -- see processorsForConfig for
	// details. The upshot is that, for now, if the input configuration specifies
	// processors, they must be returned as direct children of the resulting
	// processors.Processors (rather than being collected in additional tree
	// structure).
	// This test should be removed once we have a more consistent mechanism for
	// collecting and running processors.
	configStr := `processors:
- add_fields: {fields: {testField: value}}
- add_fields: {fields: {testField2: stuff}}`
	config, err := conf.NewConfigFrom(configStr)
	if err != nil {
		t.Fatal(err)
	}

	editor, sharedProcs, err := newCommonConfigEditor(beat.Info{}, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sharedProcs.Close() })

	clientCfg, err := editor(beat.ClientConfig{})
	require.NoError(t, err)

	lst := clientCfg.Processing.Processor
	assert.Len(t, lst.(*processors.Processors).List, 2) //nolint:errcheck //Safe to ignore in tests
}

// setRawIndex is a bare-bones processor to set the raw_index field to a
// constant string in the event metadata. It is used to test order of operations
// for processorsForConfig.
type setRawIndex struct {
	indexStr string
}

func (p *setRawIndex) Run(event *beat.Event) (*beat.Event, error) {
	if event.Meta == nil {
		event.Meta = mapstr.M{}
	}
	event.Meta[events.FieldMetaRawIndex] = p.indexStr
	return event, nil
}

func (p *setRawIndex) String() string {
	return fmt.Sprintf("set_raw_index=%v", p.indexStr)
}

// makeProcessors wraps one or more bare Processor objects in Processors.
func makeProcessors(procs ...beat.Processor) *processors.Processors {
	logger, _ := logp.NewDevelopmentLogger("")
	procList := processors.NewList(logger)
	procList.List = procs
	return procList
}

func TestRunnerFactoryWithCommonInputSettings(t *testing.T) {

	// we use `add_kubernetes_metadata` and `add_cloud_metadata`
	// for testing because initially the problem we've discovered
	// was visible with these 2 processors.
	configYAML := `
processors:
  - add_kubernetes_metadata: ~
  - add_cloud_metadata: ~
keep_null: true
publisher_pipeline:
  disable_host: true
type: "filestream"
service.type: "module"
pipeline: "test"
index: "%{[fields.log_type]}-%{[agent.version]}-%{+yyyy.MM.dd}"
`
	// illumos: this specific test requires add_kubernetes_metadata for side-effects
	//   in this test which trigger issues for the stubbed version provided for
	//   illumos (see prior comment about the side-effects being the purpose).
	if runtime.GOOS == "illumos" {
		configYAML = strings.ReplaceAll(configYAML, "\n  - add_kubernetes_metadata: ~", "")
	}
	cfg, err := conf.NewConfigWithYAML([]byte(configYAML), configYAML)
	require.NoError(t, err)

	b := beat.Info{Logger: logptest.NewTestingLogger(t, "")} // not important for the test
	rf := &runnerFactoryMock{
		clientCount: 3, // we will create 3 clients from the wrapped pipeline
	}
	pcm := &pipelineConnectorMock{} // creates mock pipeline clients and will get wrapped

	rfwc := RunnerFactoryWithCommonInputSettings(b, rf)

	// create a wrapped runner, our mock runner will
	// create the given amount of clients here using the wrapped pipeline connector.
	runner, err := rfwc.Create(pcm, cfg)
	require.NoError(t, err)
	t.Cleanup(runner.Stop)

	rf.Assert(t)
}

// TestSharedProcessorsAcrossClients verifies that, after the hoisting fix,
// the user-configured processors and the index processor are constructed
// once per input and shared across all clients connected to the wrapped
// pipeline — instead of being instantiated per client/harvester (the
// blow-up reported in elastic/beats#50376).
func TestSharedProcessorsAcrossClients(t *testing.T) {
	configYAML := `
processors:
  - add_fields: {fields: {testField: a}}
  - add_fields: {fields: {testField2: b}}
index: "static-index"
`
	cfg, err := conf.NewConfigWithYAML([]byte(configYAML), configYAML)
	require.NoError(t, err)

	b := beat.Info{Logger: logptest.NewTestingLogger(t, "")}

	editor, sharedProcs, err := newCommonConfigEditor(b, cfg)
	require.NoError(t, err)
	require.NotNil(t, sharedProcs)
	t.Cleanup(func() { _ = sharedProcs.Close() })

	// 2 add_fields + 1 index processor = 3
	require.Len(t, sharedProcs.List, 3)

	const numClients = 4
	collected := make([][]beat.Processor, 0, numClients)
	for i := 0; i < numClients; i++ {
		clientCfg, err := editor(beat.ClientConfig{})
		require.NoError(t, err)
		collected = append(collected, clientCfg.Processing.Processor.All())
	}

	assertNoCloseProcessorsShared(t, collected)
}

// assertNoCloseProcessorsShared verifies that for every pair of per-client
// processor lists, each entry is a *noCloseProcessor whose wrapper is
// distinct per client (so per-client Close is isolated) but whose inner
// instance is identical (the shared per-input processor — #50376).
func assertNoCloseProcessorsShared(t *testing.T, perClient [][]beat.Processor) {
	t.Helper()
	require.NotEmpty(t, perClient, "need at least one client list")
	require.NotEmpty(t, perClient[0], "client 0 list cannot be empty")
	for i := 1; i < len(perClient); i++ {
		require.Lenf(t, perClient[i], len(perClient[0]), "client %d list length differs from client 0", i)
		for j := range perClient[i] {
			w0, ok := perClient[0][j].(*noCloseProcessor)
			require.Truef(t, ok, "client 0 processor[%d] expected *noCloseProcessor, got %T", j, perClient[0][j])
			wi, ok := perClient[i][j].(*noCloseProcessor)
			require.Truef(t, ok, "client %d processor[%d] expected *noCloseProcessor, got %T", i, j, perClient[i][j])
			require.NotSamef(t, w0, wi, "client %d processor[%d]: wrappers must be distinct allocations", i, j)
			require.Samef(t, w0.inner, wi.inner, "client %d processor[%d]: inner shared instance must be identical to client 0's (#50376)", i, j)
		}
	}
}

// TestNoCloseProcessor verifies that the per-client wrapper:
//   - does NOT propagate Close to the shared inner processor
//   - DOES forward SetPaths so lazy-init processors (cache, script,
//     conditional processors with path-aware children) work correctly.
func TestNoCloseProcessor(t *testing.T) {
	inner := &recordingProcessor{}
	w := &noCloseProcessor{inner: inner}

	// Close path: a per-client list closing the wrapper must not close inner.
	require.NoError(t, processors.Close(w))
	require.Falsef(t, inner.closed, "inner processor must not be closed via the wrapper")

	// SetPaths forwarding: the wrapper must implement PathSetter and delegate.
	ps, ok := any(w).(processors.PathSetter)
	require.Truef(t, ok, "noCloseProcessor must implement PathSetter so the publisher pipeline's group.SetPaths reaches the inner processor")
	require.NoError(t, ps.SetPaths(nil))
	require.Equal(t, 1, inner.setPathsCalls)

	// Run forwarding: events flow through to inner.
	ev := &beat.Event{Fields: mapstr.M{}}
	out, err := w.Run(ev)
	require.NoError(t, err)
	require.Same(t, ev, out)
	require.Equal(t, 1, inner.runCalls)
}

// recordingProcessor implements beat.Processor, processors.Closer, and
// processors.PathSetter to observe what the wrapper forwards or suppresses.
type recordingProcessor struct {
	closed        bool
	runCalls      int
	setPathsCalls int
}

func (r *recordingProcessor) Run(ev *beat.Event) (*beat.Event, error) {
	r.runCalls++
	return ev, nil
}

func (r *recordingProcessor) String() string { return "recordingProcessor" }

func (r *recordingProcessor) Close() error {
	r.closed = true
	return nil
}

// SetPaths is intentionally permissive (any *paths.Path, including nil).
func (r *recordingProcessor) SetPaths(_ *paths.Path) error {
	r.setPathsCalls++
	return nil
}

// TestRunnerWithSharedProcessorsClosesProcessorsAtStop verifies the new
// runner.Stop() ordering: the wrapped Runner stops first (draining all
// pipeline clients), THEN the shared processors are closed. Stop is also
// safe to call more than once.
func TestRunnerWithSharedProcessorsClosesProcessorsAtStop(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	inner1 := &recordingProcessor{}
	inner2 := &recordingProcessor{}

	procs := processors.NewList(logger)
	procs.AddProcessor(inner1)
	procs.AddProcessor(inner2)

	stopped := 0
	r := &runnerWithSharedProcessors{
		Runner: &stopOrderRunner{onStop: func() {
			stopped++
			// Inner must still be live when the underlying Runner is
			// stopping — clients may still be flushing on Stop().
			require.Falsef(t, inner1.closed, "shared processor closed before Runner.Stop returned")
			require.Falsef(t, inner2.closed, "shared processor closed before Runner.Stop returned")
		}},
		procs: procs,
	}

	r.Stop()
	require.Equal(t, 1, stopped)
	require.True(t, inner1.closed, "shared processor must be closed after Runner.Stop returns")
	require.True(t, inner2.closed, "shared processor must be closed after Runner.Stop returns")

	// Idempotent: a second Stop must not call the underlying Stop again
	// and must not error.
	r.Stop()
	require.Equal(t, 1, stopped, "Stop must be idempotent")
}

type stopOrderRunner struct {
	onStop func()
}

func (s *stopOrderRunner) Start()         {}
func (s *stopOrderRunner) Stop()          { s.onStop() }
func (s *stopOrderRunner) String() string { return "stopOrderRunner" }

// TestRunnerWithSharedProcessorsForwardsStatusReporter verifies the wrapper
// exposes SetStatusReporter to runtime type-assertion callers (used by
// libbeat/cfgfile/list.go to wire elastic-agent-client status reporting)
// and is a no-op when the inner runner does not implement it.
func TestRunnerWithSharedProcessorsForwardsStatusReporter(t *testing.T) {
	inner := &statusReporterRunner{}
	r := &runnerWithSharedProcessors{
		Runner: inner,
		procs:  processors.NewList(logptest.NewTestingLogger(t, "")),
	}

	sr, ok := any(r).(status.WithStatusReporter)
	require.Truef(t, ok, "runnerWithSharedProcessors must implement status.WithStatusReporter")

	reporter := &recordingStatusReporter{}
	sr.SetStatusReporter(reporter)
	require.Same(t, reporter, inner.lastReporter, "SetStatusReporter must reach the inner runner")

	// Inner without WithStatusReporter must not panic.
	r2 := &runnerWithSharedProcessors{
		Runner: &stopOrderRunner{onStop: func() {}},
		procs:  processors.NewList(logptest.NewTestingLogger(t, "")),
	}
	r2.SetStatusReporter(&recordingStatusReporter{})
}

type statusReporterRunner struct {
	lastReporter status.StatusReporter
}

func (s *statusReporterRunner) Start()         {}
func (s *statusReporterRunner) Stop()          {}
func (s *statusReporterRunner) String() string { return "statusReporterRunner" }
func (s *statusReporterRunner) SetStatusReporter(reporter status.StatusReporter) {
	s.lastReporter = reporter
}

type recordingStatusReporter struct{}

func (*recordingStatusReporter) UpdateStatus(status.Status, string) {}

// TestRunnerWithSharedProcessorsForwardsSetOnce verifies the wrapper
// forwards SetOnce to an inner runner that implements OnceSetter (used by
// crawler.startInput for `filebeat --once`) and is a no-op otherwise.
func TestRunnerWithSharedProcessorsForwardsSetOnce(t *testing.T) {
	inner := &onceSetterRunner{}
	r := &runnerWithSharedProcessors{
		Runner: inner,
		procs:  processors.NewList(logptest.NewTestingLogger(t, "")),
	}

	o, ok := any(r).(OnceSetter)
	require.Truef(t, ok, "runnerWithSharedProcessors must implement OnceSetter")
	o.SetOnce(true)
	require.True(t, inner.once, "SetOnce must reach the inner runner")

	// Inner without OnceSetter must not panic.
	r2 := &runnerWithSharedProcessors{
		Runner: &stopOrderRunner{onStop: func() {}},
		procs:  processors.NewList(logptest.NewTestingLogger(t, "")),
	}
	r2.SetOnce(true)
}

type onceSetterRunner struct {
	once bool
}

func (*onceSetterRunner) Start()              {}
func (*onceSetterRunner) Stop()               {}
func (*onceSetterRunner) String() string      { return "onceSetterRunner" }
func (o *onceSetterRunner) SetOnce(once bool) { o.once = once }
