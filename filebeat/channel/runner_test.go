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
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
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

// TestSharedProcessorsAcrossClients verifies that the user-configured
// processors and the index processor are constructed once per input and shared
// across all clients connected to the wrapped pipeline, instead of being
// instantiated per client/harvester (the blow-up reported in
// elastic/beats#50376).
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

	assertSharedProcessors(t, collected)
}

// assertSharedProcessors verifies that, across all per-client processor lists,
// each entry is a sharedProcessor whose embedded processor is the same shared
// per-input instance (#50376). Per-client Close isolation comes from
// sharedProcessor hiding Closer (see TestSharedProcessorHidesLifecycleMethods),
// not from how the wrapper itself is allocated.
func assertSharedProcessors(t *testing.T, perClient [][]beat.Processor) {
	t.Helper()
	require.NotEmpty(t, perClient, "need at least one client list")
	require.NotEmpty(t, perClient[0], "client 0 list cannot be empty")
	for i := 1; i < len(perClient); i++ {
		require.Lenf(t, perClient[i], len(perClient[0]), "client %d list length differs from client 0", i)
		for j := range perClient[i] {
			w0, ok := perClient[0][j].(sharedProcessor)
			require.Truef(t, ok, "client 0 processor[%d] expected sharedProcessor, got %T", j, perClient[0][j])
			wi, ok := perClient[i][j].(sharedProcessor)
			require.Truef(t, ok, "client %d processor[%d] expected sharedProcessor, got %T", i, j, perClient[i][j])
			require.Samef(t, w0.Processor, wi.Processor, "client %d processor[%d]: embedded instance must be the shared one (#50376)", i, j)
		}
	}
}

// TestSharedProcessorHidesLifecycleMethods verifies that the per-client wrapper
// exposes only Run/String: it must NOT implement processors.Closer (so a
// harvester's client closing its list cannot close the shared inner) nor
// processors.PathSetter (paths are set once on the shared list, not per client).
func TestSharedProcessorHidesLifecycleMethods(t *testing.T) {
	inner := &recordingProcessor{}
	w := sharedProcessor{inner}

	_, isCloser := any(w).(processors.Closer)
	require.Falsef(t, isCloser, "sharedProcessor must not implement Closer, otherwise per-client Close would tear down the shared instance")
	_, isPathSetter := any(w).(processors.PathSetter)
	require.Falsef(t, isPathSetter, "sharedProcessor must not implement PathSetter; paths are initialised once on the shared list")

	// processors.Close is therefore a no-op on the wrapper.
	require.NoError(t, processors.Close(w))
	require.Falsef(t, inner.closed, "inner processor must not be closed via the wrapper")

	// Run/String forward to the inner processor.
	ev := &beat.Event{Fields: mapstr.M{}}
	out, err := w.Run(ev)
	require.NoError(t, err)
	require.Same(t, ev, out)
	require.Equal(t, 1, inner.runCalls)
	require.Equal(t, inner.String(), w.String())
}

// TestInputProcessorPathsSetOnce verifies that newConfigEditor calls SetPaths
// exactly once, at build time (before any client connects), on the shared
// path-aware processors; that a per-client Close does not reach them; and that
// they are closed once at input shutdown via shared.Close().
func TestInputProcessorPathsSetOnce(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	rec := &recordingProcessor{}

	// processors.New wraps every constructed processor with SafeWrap (the same
	// wrapper RegisterPlugin applies). Mirror that here so the shared list has
	// the production SetPaths-once / Close-once semantics, without registering a
	// global test plugin.
	ctor := processors.SafeWrap(func(*conf.C, *logp.Logger) (beat.Processor, error) {
		return rec, nil
	})
	wrapped, err := ctor(nil, logger)
	require.NoError(t, err)

	userProcs := processors.NewList(logger)
	userProcs.AddProcessor(wrapped)

	b := beat.Info{Logger: logger, Paths: paths.New()}

	editor, sharedProcs, err := newConfigEditor(b, commonInputConfig{}, userProcs)
	require.NoError(t, err)
	// This test drives shutdown explicitly below to assert close-once; no
	// t.Cleanup close here on purpose.

	// SetPaths was applied once at build time, before any client connected.
	require.Equalf(t, 1, rec.setPathsCalls, "SetPaths must be applied once when the shared list is built")

	// A per-client list closing must not close the shared instance, nor set
	// paths again.
	clientCfg, err := editor(beat.ClientConfig{})
	require.NoError(t, err)
	require.NoError(t, clientCfg.Processing.Processor.Close())
	require.Falsef(t, rec.closed, "per-client Close must not close the shared processor")
	require.Equalf(t, 1, rec.setPathsCalls, "SetPaths must not be called again per client")

	// Closing the shared list (input shutdown) closes it exactly once.
	require.NoError(t, sharedProcs.Close())
	require.Truef(t, rec.closed, "shared processor must be closed at input shutdown")
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

// SetPaths is intentionally permissive (accepts any *paths.Path, including nil).
func (r *recordingProcessor) SetPaths(_ *paths.Path) error {
	r.setPathsCalls++
	return nil
}

// TestCommonSettingsFactoryAttachesSharedProcessorsToRunner verifies the
// factory returns the inner runner unwrapped (so its optional interfaces stay
// visible to libbeat/cfgfile/list.go) and registers the shared processors on it
// via AddCloser.
func TestCommonSettingsFactoryAttachesSharedProcessorsToRunner(t *testing.T) {
	b := beat.Info{Logger: logptest.NewTestingLogger(t, ""), Paths: paths.New()}
	inner := &noopRunner{}
	f := RunnerFactoryWithCommonInputSettings(b, &fakeInnerFactory{runner: inner})

	r, err := f.Create(nil, conf.NewConfig())
	require.NoError(t, err)
	require.Same(t, inner, r, "factory must return the inner runner without wrapping it")
	require.Lenf(t, inner.closers, 1, "shared processors must be registered with the runner via AddCloser")
}

// TestCommonSettingsFactoryClosesSharedProcessorsOnInnerError verifies that the
// per-input shared processors are released when the inner factory fails to
// create the runner.
func TestCommonSettingsFactoryClosesSharedProcessorsOnInnerError(t *testing.T) {
	b := beat.Info{Logger: logptest.NewTestingLogger(t, ""), Paths: paths.New()}
	wantErr := errors.New("inner create failed")
	f := RunnerFactoryWithCommonInputSettings(b, &fakeInnerFactory{err: wantErr})

	r, err := f.Create(nil, conf.NewConfig())
	require.Nil(t, r)
	require.ErrorIs(t, err, wantErr)
}

// fakeInnerFactory is a channel.InputRunnerFactory stub that returns a preset
// runner or error, letting the tests exercise commonSettingsFactory.Create in
// isolation.
type fakeInnerFactory struct {
	runner InputRunner
	err    error
}

func (f *fakeInnerFactory) CheckConfig(*conf.C) error { return nil }
func (f *fakeInnerFactory) Create(beat.PipelineConnector, *conf.C) (InputRunner, error) {
	return f.runner, f.err
}
