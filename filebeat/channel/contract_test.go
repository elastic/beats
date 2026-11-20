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

package channel_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/processors"
	_ "github.com/elastic/beats/v7/libbeat/processors/actions"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/mapstr"
)

const chanCloserName = "contract_test_channel_closer"

var chanIDCounter atomic.Int64

func chanUniqueID(name string) string {
	return fmt.Sprintf("%s-%d", name, chanIDCounter.Add(1))
}

// chanCloserState records calls received by an underlying mock processor.
type chanCloserState struct {
	id    string
	field string
	value string

	mu             sync.Mutex
	closed         bool
	closeCount     int
	runsAfterClose int
}

func (p *chanCloserState) Run(event *beat.Event) (*beat.Event, error) {
	p.mu.Lock()
	if p.closed {
		p.runsAfterClose++
		p.mu.Unlock()
		return nil, fmt.Errorf("contract mock processor %q ran after Close", p.id)
	}
	p.mu.Unlock()

	if _, err := event.PutValue(p.field, p.value); err != nil {
		return nil, err
	}
	return event, nil
}

func (p *chanCloserState) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closeCount++
	p.closed = true
	return nil
}

func (p *chanCloserState) String() string {
	return fmt.Sprintf("contract_test_channel_mock(id=%s)", p.id)
}

type chanTracker struct {
	mu        sync.Mutex
	instances map[string][]*chanCloserState
}

var chanInstances = &chanTracker{instances: map[string][]*chanCloserState{}}

func (tr *chanTracker) add(st *chanCloserState) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.instances[st.id] = append(tr.instances[st.id], st)
}

func (tr *chanTracker) get(id string) []*chanCloserState {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return append([]*chanCloserState(nil), tr.instances[id]...)
}

func init() {
	processors.RegisterPlugin(chanCloserName,
		func(cfg *conf.C, _ *logp.Logger) (beat.Processor, error) {
			c := struct {
				ID    string `config:"id"`
				Field string `config:"field"`
				Value string `config:"value"`
			}{}
			if cfg != nil {
				if err := cfg.Unpack(&c); err != nil {
					return nil, err
				}
			}
			if c.ID == "" || c.Field == "" {
				return nil, errors.New("contract_test_channel mock: 'id' and 'field' are required")
			}
			st := &chanCloserState{id: c.ID, field: c.Field, value: c.Value}
			chanInstances.add(st)
			return st, nil
		})
}

type nopRunner struct{}

func (nopRunner) String() string { return "contract-test-runner" }
func (nopRunner) Start()         {}
func (nopRunner) Stop()          {}

// connectorCapturingFactory exposes the connector passed to a runner.
type connectorCapturingFactory struct {
	mu         sync.Mutex
	connectors []beat.PipelineConnector
}

func (f *connectorCapturingFactory) Create(p beat.PipelineConnector, _ *conf.C) (cfgfile.Runner, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.connectors = append(f.connectors, p)
	return nopRunner{}, nil
}

func (f *connectorCapturingFactory) CheckConfig(*conf.C) error { return nil }

func (f *connectorCapturingFactory) last(t testing.TB) beat.PipelineConnector {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	require.NotEmpty(t, f.connectors)
	return f.connectors[len(f.connectors)-1]
}

type recordedClient struct{ cfg beat.ClientConfig }

func (recordedClient) Publish(beat.Event)      {}
func (recordedClient) PublishAll([]beat.Event) {}
func (recordedClient) Close() error            { return nil }

type stubPipeline struct{}

func (stubPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	return &recordedClient{cfg: cfg}, nil
}

func (p stubPipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

func (stubPipeline) Disconnect(context.Context) error { return nil }

func chanBeatInfo(t testing.TB) beat.Info {
	return beat.Info{
		Beat:    "contractbeat",
		Version: "9.9.9",
		Logger:  logptest.NewTestingLogger(t, ""),
	}
}

func chanInputCfg(id, value string) mapstr.M {
	return mapstr.M{
		"type":  "filestream",
		"index": "contract-index",
		"processors": []mapstr.M{
			{chanCloserName: mapstr.M{"id": id, "field": "marker", "value": value}},
			{"add_fields": mapstr.M{"target": "", "fields": mapstr.M{"static": "static-value"}}},
		},
	}
}

func createInputConnector(t testing.TB, raw mapstr.M) beat.PipelineConnector {
	t.Helper()
	cfg, err := conf.NewConfigFrom(raw)
	require.NoError(t, err)

	factory := &connectorCapturingFactory{}
	rfwc := channel.RunnerFactoryWithCommonInputSettings(chanBeatInfo(t), factory)
	_, err = rfwc.Create(stubPipeline{}, cfg)
	require.NoError(t, err)
	return factory.last(t)
}

func connectProcessors(t testing.TB, connector beat.PipelineConnector) beat.ProcessorList {
	t.Helper()
	client, err := connector.ConnectWith(beat.ClientConfig{})
	require.NoError(t, err)
	rc, ok := client.(*recordedClient)
	require.True(t, ok)
	require.NotNil(t, rc.cfg.Processing.Processor)
	return rc.cfg.Processing.Processor
}

// connectProcessing returns the processing settings the input passed to the
// pipeline for a client connecting with processing.
func connectProcessing(t *testing.T, connector beat.PipelineConnector, processing beat.ProcessingConfig) beat.ProcessingConfig {
	t.Helper()
	client, err := connector.ConnectWith(beat.ClientConfig{Processing: processing})
	require.NoError(t, err)
	rc, ok := client.(*recordedClient)
	require.Truef(t, ok, "client is %T, want *recordedClient", client)
	t.Cleanup(func() { assert.NoError(t, rc.cfg.Processing.Processor.Close()) })
	return rc.cfg.Processing
}

func sameMap(a, b mapstr.M) bool {
	return reflect.ValueOf(a).UnsafePointer() == reflect.ValueOf(b).UnsafePointer()
}

func chanEvent() *beat.Event {
	return &beat.Event{
		Timestamp: time.Now(),
		Fields:    mapstr.M{"message": "contract-test"},
	}
}

func chanRequireMarker(t *testing.T, evt *beat.Event, err error, field, want string) {
	t.Helper()
	require.NoError(t, err)
	require.NotNil(t, evt)
	got, gerr := evt.GetValue(field)
	require.NoError(t, gerr, "field %q missing from event", field)
	require.Equal(t, want, got)
}

// A Run racing with or after Close may fail or return a correctly processed
// event, but it must not silently drop or misprocess the event.
func chanCheckErrOrCorrect(evt *beat.Event, err error, field, want string) error {
	if err != nil {
		return nil //nolint:nilerr // an error result is an acceptable outcome here
	}
	if evt == nil {
		return errors.New("event was dropped without an error")
	}
	got, gerr := evt.GetValue(field)
	if gerr != nil {
		return fmt.Errorf("run succeeded but marker field %q is missing: %w", field, gerr)
	}
	if got != want {
		return fmt.Errorf("marker field %q has value %v, want %q", field, got, want)
	}
	return nil
}

func chanAssertLifecycleInvariants(t *testing.T, id string) {
	t.Helper()
	for i, st := range chanInstances.get(id) {
		st.mu.Lock()
		closeCount, runsAfterClose := st.closeCount, st.runsAfterClose
		st.mu.Unlock()
		assert.LessOrEqualf(t, closeCount, 1,
			"instance %d of %q: underlying Close must be called at most once per constructed instance", i, id)
		assert.Zerof(t, runsAfterClose,
			"instance %d of %q: underlying Run must never be called after Close", i, id)
	}
}

func chanAssertAllClosed(t *testing.T, id string) {
	t.Helper()
	insts := chanInstances.get(id)
	require.NotEmptyf(t, insts, "no mock instances were constructed for id %q", id)
	for i, st := range insts {
		st.mu.Lock()
		closed := st.closed
		st.mu.Unlock()
		assert.Truef(t, closed,
			"instance %d of %q: closing all owning clients' processors must close the processor", i, id)
	}
}

func TestContractInputProcessorConstruction(t *testing.T) {
	id := chanUniqueID("input-basic")
	connector := createInputConnector(t, chanInputCfg(id, "input-value"))
	procs := connectProcessors(t, connector)

	evt, err := procs.Run(chanEvent())
	chanRequireMarker(t, evt, err, "marker", "input-value")
	chanRequireMarker(t, evt, err, "static", "static-value")
	chanRequireMarker(t, evt, err, "@metadata.raw_index", "contract-index")

	assert.NoError(t, procs.Close())
	chanAssertAllClosed(t, id)
	assert.NoError(t, procs.Close())
	chanAssertLifecycleInvariants(t, id)

	revt, rerr := procs.Run(chanEvent())
	assert.NoError(t, chanCheckErrOrCorrect(revt, rerr, "marker", "input-value"))
}

func TestContractInputConstructionErrorPropagation(t *testing.T) {
	t.Run("unknown processor", func(t *testing.T) {
		raw := mapstr.M{
			"processors": []mapstr.M{
				{"contract_test_channel_does_not_exist": mapstr.M{}},
			},
		}
		cfg, err := conf.NewConfigFrom(raw)
		require.NoError(t, err)

		factory := &connectorCapturingFactory{}
		rfwc := channel.RunnerFactoryWithCommonInputSettings(chanBeatInfo(t), factory)
		_, err = rfwc.Create(stubPipeline{}, cfg)
		if err == nil {
			// The error may surface when the client connects instead.
			_, err = factory.last(t).ConnectWith(beat.ClientConfig{})
		}
		assert.ErrorContains(t, err, "contract_test_channel_does_not_exist")
	})

	t.Run("failing processor configuration", func(t *testing.T) {
		raw := mapstr.M{
			"processors": []mapstr.M{
				{chanCloserName: mapstr.M{"value": "no-id-no-field"}},
			},
		}
		cfg, err := conf.NewConfigFrom(raw)
		require.NoError(t, err)

		factory := &connectorCapturingFactory{}
		rfwc := channel.RunnerFactoryWithCommonInputSettings(chanBeatInfo(t), factory)
		_, err = rfwc.Create(stubPipeline{}, cfg)
		if err == nil {
			_, err = factory.last(t).ConnectWith(beat.ClientConfig{})
		}
		assert.ErrorContains(t, err, "'id' and 'field' are required")
	})
}

func TestContractInputClientIndependence(t *testing.T) {
	t.Run("clients of the same input", func(t *testing.T) {
		id := chanUniqueID("same-input")
		connector := createInputConnector(t, chanInputCfg(id, "same-input-value"))

		procsA := connectProcessors(t, connector)
		procsB := connectProcessors(t, connector)

		evt, err := procsA.Run(chanEvent())
		chanRequireMarker(t, evt, err, "marker", "same-input-value")
		evt, err = procsB.Run(chanEvent())
		chanRequireMarker(t, evt, err, "marker", "same-input-value")

		assert.NoError(t, procsA.Close())

		for range 3 {
			evt, err = procsB.Run(chanEvent())
			chanRequireMarker(t, evt, err, "marker", "same-input-value")
			chanRequireMarker(t, evt, err, "static", "static-value")
		}

		assert.NoError(t, procsB.Close())
		chanAssertAllClosed(t, id)
		chanAssertLifecycleInvariants(t, id)
	})

	t.Run("separate inputs with identical configuration", func(t *testing.T) {
		id := chanUniqueID("identical-inputs")
		connectorA := createInputConnector(t, chanInputCfg(id, "identical-value"))
		connectorB := createInputConnector(t, chanInputCfg(id, "identical-value"))

		procsA := connectProcessors(t, connectorA)
		procsB := connectProcessors(t, connectorB)

		evt, err := procsA.Run(chanEvent())
		chanRequireMarker(t, evt, err, "marker", "identical-value")

		assert.NoError(t, procsA.Close())

		for range 3 {
			evt, err = procsB.Run(chanEvent())
			chanRequireMarker(t, evt, err, "marker", "identical-value")
		}

		assert.NoError(t, procsB.Close())
		chanAssertAllClosed(t, id)
		chanAssertLifecycleInvariants(t, id)
	})

	t.Run("separate inputs with different configurations", func(t *testing.T) {
		idA := chanUniqueID("input-a")
		idB := chanUniqueID("input-b")
		procsA := connectProcessors(t, createInputConnector(t, chanInputCfg(idA, "value-a")))
		procsB := connectProcessors(t, createInputConnector(t, chanInputCfg(idB, "value-b")))

		evt, err := procsA.Run(chanEvent())
		chanRequireMarker(t, evt, err, "marker", "value-a")
		evt, err = procsB.Run(chanEvent())
		chanRequireMarker(t, evt, err, "marker", "value-b")

		assert.NoError(t, procsA.Close())

		for range 3 {
			evt, err = procsB.Run(chanEvent())
			chanRequireMarker(t, evt, err, "marker", "value-b")
		}

		assert.NoError(t, procsB.Close())
		for _, id := range []string{idA, idB} {
			chanAssertAllClosed(t, id)
			chanAssertLifecycleInvariants(t, id)
		}
	})
}

func TestContractInputConcurrentHarvesters(t *testing.T) {
	const (
		workers      = 10
		harvesters   = 3
		eventsPerRun = 8
	)

	info := chanBeatInfo(t)

	sharedID := chanUniqueID("conc-shared-input")
	const sharedValue = "conc-shared-value"

	ids := []string{sharedID}
	workerID := make([]string, workers)
	workerValue := make([]string, workers)
	for w := range workers {
		if w%2 == 0 {
			workerID[w], workerValue[w] = sharedID, sharedValue
		} else {
			workerID[w] = chanUniqueID(fmt.Sprintf("conc-input-%d", w))
			workerValue[w] = fmt.Sprintf("conc-value-%d", w)
			ids = append(ids, workerID[w])
		}
	}

	var wg sync.WaitGroup
	for w := range workers {
		id, value := workerID[w], workerValue[w]
		wg.Go(func() {
			cfg, err := conf.NewConfigFrom(chanInputCfg(id, value))
			if err != nil {
				t.Errorf("worker %d: config: %v", w, err)
				return
			}
			factory := &connectorCapturingFactory{}
			rfwc := channel.RunnerFactoryWithCommonInputSettings(info, factory)
			if _, err := rfwc.Create(stubPipeline{}, cfg); err != nil {
				t.Errorf("worker %d: Create: %v", w, err)
				return
			}
			factory.mu.Lock()
			connector := factory.connectors[0]
			factory.mu.Unlock()

			for range harvesters {
				client, err := connector.ConnectWith(beat.ClientConfig{})
				if err != nil {
					t.Errorf("worker %d: ConnectWith: %v", w, err)
					return
				}
				rc, ok := client.(*recordedClient)
				if !ok || rc.cfg.Processing.Processor == nil {
					t.Errorf("worker %d: client has no processors", w)
					return
				}
				procs := rc.cfg.Processing.Processor

				for range eventsPerRun {
					evt, err := procs.Run(chanEvent())
					if err != nil {
						t.Errorf("worker %d: Run: %v", w, err)
						continue
					}
					if evt == nil {
						t.Errorf("worker %d: Run dropped the event", w)
						continue
					}
					got, gerr := evt.GetValue("marker")
					if gerr != nil {
						t.Errorf("worker %d: marker missing: %v", w, gerr)
						continue
					}
					if got != value {
						t.Errorf("worker %d: marker = %v, want %q (event processed with another input's config)",
							w, got, value)
					}
				}

				if err := procs.Close(); err != nil {
					t.Errorf("worker %d: Close: %v", w, err)
				}
				if err := procs.Close(); err != nil {
					t.Errorf("worker %d: double Close: %v", w, err)
				}
				evt, rerr := procs.Run(chanEvent())
				if err := chanCheckErrOrCorrect(evt, rerr, "marker", value); err != nil {
					t.Errorf("worker %d: Run after Close: %v", w, err)
				}
			}
		})
	}
	wg.Wait()

	for _, id := range ids {
		chanAssertAllClosed(t, id)
		chanAssertLifecycleInvariants(t, id)
	}
}

func TestContractInputClientsShareFields(t *testing.T) {
	filestream := mapstr.M{"type": "filestream"}
	cases := map[string]struct {
		input        mapstr.M
		client       beat.ProcessingConfig
		wantFields   mapstr.M
		wantReadOnly bool
	}{
		"no client fields": {
			input:        filestream,
			wantFields:   mapstr.M{"input": mapstr.M{"type": "filestream"}},
			wantReadOnly: true,
		},
		"module input": {
			input: mapstr.M{"type": "log", "_module_name": "nginx", "_fileset_name": "access"},
			wantFields: mapstr.M{
				"input":   mapstr.M{"type": "log"},
				"fileset": mapstr.M{"name": "access"},
				"service": mapstr.M{"type": "nginx"},
				"event":   mapstr.M{"module": "nginx", "dataset": "nginx.access"},
			},
			wantReadOnly: true,
		},
		"type disabled": {
			input:      filestream,
			client:     beat.ProcessingConfig{DisableType: true},
			wantFields: mapstr.M{},
		},
		"client fields": {
			input:      filestream,
			client:     beat.ProcessingConfig{Fields: mapstr.M{"client": "c"}},
			wantFields: mapstr.M{"client": "c", "input": mapstr.M{"type": "filestream"}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			connector := createInputConnector(t, tc.input)
			a := connectProcessing(t, connector, tc.client)
			b := connectProcessing(t, connector, tc.client)

			for _, got := range []beat.ProcessingConfig{a, b} {
				assert.Equal(t, tc.wantFields, got.Fields)
				assert.Equal(t, tc.wantReadOnly, got.FieldsReadOnly, "FieldsReadOnly")
			}
			assert.Equal(t, tc.wantReadOnly, sameMap(a.Fields, b.Fields), "clients of one input share the fields map")
		})
	}

	t.Run("separate inputs", func(t *testing.T) {
		a := connectProcessing(t, createInputConnector(t, filestream), beat.ProcessingConfig{})
		b := connectProcessing(t, createInputConnector(t, filestream), beat.ProcessingConfig{})
		assert.False(t, sameMap(a.Fields, b.Fields), "separate inputs share the fields map")
	})
}

// TestContractInputSharedFieldsWithProcessing runs events of two clients of one
// input through chains built by the processing builder.
func TestContractInputSharedFieldsWithProcessing(t *testing.T) {
	cases := map[string]struct {
		input     mapstr.M
		wantInput mapstr.M
	}{
		"no processors": {
			input:     mapstr.M{"type": "filestream"},
			wantInput: mapstr.M{"type": "filestream"},
		},
		"processor writing into the input fields": {
			input: mapstr.M{
				"type": "filestream",
				"processors": []mapstr.M{
					{"add_fields": mapstr.M{"target": "input", "fields": mapstr.M{"extra": "x"}}},
				},
			},
			wantInput: mapstr.M{"type": "filestream", "extra": "x"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			support, err := processing.MakeDefaultSupport(true, nil)(chanBeatInfo(t), logptest.NewTestingLogger(t, ""), conf.NewConfig())
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, support.Close()) })
			connector := createInputConnector(t, tc.input)

			for c := range 2 {
				cfg := connectProcessing(t, connector, beat.ProcessingConfig{})
				chain, err := support.Create(cfg, false)
				require.NoError(t, err)

				for range 2 {
					out, err := chain.Run(chanEvent())
					require.NoError(t, err)
					require.NotNil(t, out)
					assert.Equalf(t, mapstr.M{"message": "contract-test", "input": tc.wantInput}, out.Fields, "client %d event", c)
				}
				assert.Equalf(t, mapstr.M{"input": mapstr.M{"type": "filestream"}}, cfg.Fields,
					"client %d fields after its events ran", c)
			}
		})
	}
}
