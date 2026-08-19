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

package processors_test

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v9/libbeat/beat"
	"github.com/elastic/beats/v9/libbeat/processors"
	_ "github.com/elastic/beats/v9/libbeat/processors/actions"
	"github.com/elastic/beats/v9/libbeat/processors/actions/addfields"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/paths"
)

const (
	contractCloserName     = "contract_test_closer"
	contractPathSetterName = "contract_test_pathsetter"
	contractFailingName    = "contract_test_failing"
)

const contractConstructorFailure = "intentional contract_test constructor failure"

var contractIDCounter atomic.Int64

func contractUniqueID(name string) string {
	return fmt.Sprintf("%s-%d", name, contractIDCounter.Add(1))
}

type contractProcSnapshot struct {
	closed             bool
	closeCount         int
	runsAfterClose     int
	runsBeforeSetPaths int
	setPathsCount      int
	setPathsAfterClose int
	pathsSeen          []paths.Path
}

// contractProcState records calls received by an underlying mock processor.
type contractProcState struct {
	id        string
	field     string
	value     string
	pathAware bool

	mu                 sync.Mutex
	closed             bool
	closeCount         int
	runsAfterClose     int
	runsBeforeSetPaths int
	setPathsCount      int
	setPathsAfterClose int
	pathsSeen          []paths.Path
}

func (p *contractProcState) Run(event *beat.Event) (*beat.Event, error) {
	p.mu.Lock()
	if p.closed {
		p.runsAfterClose++
		p.mu.Unlock()
		return nil, fmt.Errorf("contract mock processor %q ran after Close", p.id)
	}
	if p.pathAware && len(p.pathsSeen) == 0 {
		p.runsBeforeSetPaths++
		p.mu.Unlock()
		return nil, fmt.Errorf("contract mock processor %q ran before SetPaths", p.id)
	}
	p.mu.Unlock()

	if _, err := event.PutValue(p.field, p.value); err != nil {
		return nil, err
	}
	return event, nil
}

func (p *contractProcState) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closeCount++
	p.closed = true
	return nil
}

func (p *contractProcState) String() string {
	return fmt.Sprintf("contract_test_mock(id=%s)", p.id)
}

func (p *contractProcState) snapshot() contractProcSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	return contractProcSnapshot{
		closed:             p.closed,
		closeCount:         p.closeCount,
		runsAfterClose:     p.runsAfterClose,
		runsBeforeSetPaths: p.runsBeforeSetPaths,
		setPathsCount:      p.setPathsCount,
		setPathsAfterClose: p.setPathsAfterClose,
		pathsSeen:          append([]paths.Path(nil), p.pathsSeen...),
	}
}

type contractCloserProcessor struct{ *contractProcState }

type contractPathSetterProcessor struct{ *contractProcState }

func (p *contractPathSetterProcessor) SetPaths(pp *paths.Path) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		p.setPathsAfterClose++
		return fmt.Errorf("contract mock processor %q: SetPaths after Close", p.id)
	}
	p.setPathsCount++
	if pp != nil {
		p.pathsSeen = append(p.pathsSeen, *pp)
	}
	return nil
}

type contractTracker struct {
	mu        sync.Mutex
	instances map[string][]*contractProcState
}

var contractInstances = &contractTracker{instances: map[string][]*contractProcState{}}

func (tr *contractTracker) add(st *contractProcState) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.instances[st.id] = append(tr.instances[st.id], st)
}

func (tr *contractTracker) get(id string) []*contractProcState {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return append([]*contractProcState(nil), tr.instances[id]...)
}

type contractMockConfig struct {
	ID    string `config:"id"`
	Field string `config:"field"`
	Value string `config:"value"`
}

func newContractProcState(cfg *conf.C, pathAware bool) (*contractProcState, error) {
	c := contractMockConfig{}
	if cfg != nil {
		if err := cfg.Unpack(&c); err != nil {
			return nil, err
		}
	}
	if c.ID == "" || c.Field == "" {
		return nil, errors.New("contract_test mock: 'id' and 'field' settings are required")
	}
	st := &contractProcState{id: c.ID, field: c.Field, value: c.Value, pathAware: pathAware}
	contractInstances.add(st)
	return st, nil
}

func init() {
	processors.RegisterPlugin(contractCloserName,
		func(cfg *conf.C, _ *logp.Logger) (beat.Processor, error) {
			st, err := newContractProcState(cfg, false)
			if err != nil {
				return nil, err
			}
			return &contractCloserProcessor{st}, nil
		})
	processors.RegisterPlugin(contractPathSetterName,
		func(cfg *conf.C, _ *logp.Logger) (beat.Processor, error) {
			st, err := newContractProcState(cfg, true)
			if err != nil {
				return nil, err
			}
			return &contractPathSetterProcessor{st}, nil
		})
	processors.RegisterPlugin(contractFailingName,
		func(cfg *conf.C, _ *logp.Logger) (beat.Processor, error) {
			return nil, errors.New(contractConstructorFailure)
		})
}

func closerCfg(id, field, value string) mapstr.M {
	return mapstr.M{
		contractCloserName: mapstr.M{"id": id, "field": field, "value": value},
	}
}

func buildContractProcessors(t testing.TB, raw ...mapstr.M) *processors.Processors {
	t.Helper()
	cfg, err := processors.NewPluginConfigFromList(raw)
	require.NoError(t, err)
	procs, err := processors.New(cfg, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	require.NotNil(t, procs)
	return procs
}

func contractEvent() *beat.Event {
	return &beat.Event{
		Timestamp: time.Now(),
		Fields:    mapstr.M{"type": "contract-test"},
	}
}

func requireMarker(t *testing.T, evt *beat.Event, err error, field, want string) {
	t.Helper()
	require.NoError(t, err)
	require.NotNil(t, evt)
	got, gerr := evt.GetValue(field)
	require.NoError(t, gerr, "field %q missing from event", field)
	require.Equal(t, want, got)
}

// A Run racing with or after Close may fail or return a correctly processed
// event, but it must not silently drop or misprocess the event.
func checkErrOrCorrect(evt *beat.Event, err error, field, want string) error {
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

func assertLifecycleInvariants(t *testing.T, id string) {
	t.Helper()
	for i, st := range contractInstances.get(id) {
		snap := st.snapshot()
		assert.LessOrEqualf(t, snap.closeCount, 1,
			"instance %d of %q: underlying Close must be called at most once per constructed instance", i, id)
		assert.Zerof(t, snap.runsAfterClose,
			"instance %d of %q: underlying Run must never be called after Close", i, id)
		assert.Zerof(t, snap.setPathsAfterClose,
			"instance %d of %q: underlying SetPaths must never be called after Close", i, id)
	}
}

func assertAllClosed(t *testing.T, id string) {
	t.Helper()
	insts := contractInstances.get(id)
	require.NotEmptyf(t, insts, "no mock instances were constructed for id %q", id)
	for i, st := range insts {
		snap := st.snapshot()
		assert.Truef(t, snap.closed,
			"instance %d of %q: closing all owning processor groups must close the processor", i, id)
	}
}

func TestContractRegistryConstruction(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	t.Run("GetConstructor returns a working constructor", func(t *testing.T) {
		id := contractUniqueID("get-constructor")

		constructor, err := processors.GetConstructor(contractCloserName)
		require.NoError(t, err)
		require.NotNil(t, constructor)

		cfg, err := conf.NewConfigFrom(mapstr.M{"id": id, "field": "marker", "value": "via-get-constructor"})
		require.NoError(t, err)

		p, err := constructor(cfg, logger)
		require.NoError(t, err)
		require.NotNil(t, p)

		evt, err := p.Run(contractEvent())
		requireMarker(t, evt, err, "marker", "via-get-constructor")

		assert.NoError(t, processors.Close(p))
		assertAllClosed(t, id)
		assert.NoError(t, processors.Close(p))
		assertLifecycleInvariants(t, id)
	})

	t.Run("GetConstructor fails for unknown processor", func(t *testing.T) {
		_, err := processors.GetConstructor("contract_test_does_not_exist")
		assert.Error(t, err)
	})

	t.Run("New fails for unknown processor and names it", func(t *testing.T) {
		cfg, err := processors.NewPluginConfigFromList([]mapstr.M{
			{"contract_test_does_not_exist": mapstr.M{}},
		})
		require.NoError(t, err)

		procs, err := processors.New(cfg, logger)
		assert.ErrorContains(t, err, "contract_test_does_not_exist")
		assert.Nil(t, procs)
	})

	t.Run("New rejects an entry with multiple actions", func(t *testing.T) {
		cfg, err := processors.NewPluginConfigFromList([]mapstr.M{
			{
				"add_fields":  mapstr.M{"target": "", "fields": mapstr.M{"a": "b"}},
				"drop_fields": mapstr.M{"fields": []string{"a"}},
			},
		})
		require.NoError(t, err)

		_, err = processors.New(cfg, logger)
		assert.Error(t, err)
	})

	t.Run("empty configuration yields a pass-through group", func(t *testing.T) {
		procs, err := processors.New(nil, logger)
		require.NoError(t, err)
		require.NotNil(t, procs)

		evt, err := procs.Run(contractEvent())
		require.NoError(t, err)
		require.NotNil(t, evt)
		typ, err := evt.GetValue("type")
		require.NoError(t, err)
		require.Equal(t, "contract-test", typ)

		assert.NoError(t, procs.Close())
	})

	t.Run("NewList yields an empty runnable group", func(t *testing.T) {
		procs := processors.NewList(logger)
		require.NotNil(t, procs)

		evt, err := procs.Run(contractEvent())
		require.NoError(t, err)
		require.NotNil(t, evt)
		assert.NoError(t, procs.Close())
	})
}

func TestContractConstructorErrorPropagation(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	t.Run("direct construction via GetConstructor", func(t *testing.T) {
		constructor, err := processors.GetConstructor(contractFailingName)
		require.NoError(t, err)

		cfg, err := conf.NewConfigFrom(mapstr.M{})
		require.NoError(t, err)

		p, err := constructor(cfg, logger)
		assert.ErrorContains(t, err, contractConstructorFailure)
		assert.Nil(t, p)
	})

	t.Run("construction via New", func(t *testing.T) {
		cfg, err := processors.NewPluginConfigFromList([]mapstr.M{
			{contractFailingName: mapstr.M{}},
		})
		require.NoError(t, err)

		procs, err := processors.New(cfg, logger)
		assert.ErrorContains(t, err, contractConstructorFailure)
		assert.Nil(t, procs)
	})

	t.Run("one failing constructor fails the whole list", func(t *testing.T) {
		id := contractUniqueID("fail-mixed")
		cfg, err := processors.NewPluginConfigFromList([]mapstr.M{
			closerCfg(id, "marker", "ok"),
			{contractFailingName: mapstr.M{}},
		})
		require.NoError(t, err)

		procs, err := processors.New(cfg, logger)
		assert.ErrorContains(t, err, contractConstructorFailure)
		assert.Nil(t, procs)
	})
}

func TestContractRunOrderAndConditions(t *testing.T) {
	t.Run("processors run in configuration order", func(t *testing.T) {
		procs := buildContractProcessors(t,
			mapstr.M{"add_fields": mapstr.M{"target": "", "fields": mapstr.M{"a": "first"}}},
			mapstr.M{"rename": mapstr.M{
				"fields":        []mapstr.M{{"from": "a", "to": "b"}},
				"fail_on_error": true,
			}},
		)
		defer procs.Close()

		evt, err := procs.Run(contractEvent())
		requireMarker(t, evt, err, "b", "first")
		_, err = evt.GetValue("a")
		assert.Error(t, err, "field 'a' should have been renamed away")
	})

	t.Run("when condition gates a registered processor", func(t *testing.T) {
		id := contractUniqueID("conditional")
		procs := buildContractProcessors(t, mapstr.M{
			contractCloserName: mapstr.M{
				"id": id, "field": "marker", "value": "matched",
				"when": mapstr.M{"equals": mapstr.M{"flag": "yes"}},
			},
		})

		matching := contractEvent()
		matching.Fields["flag"] = "yes"
		evt, err := procs.Run(matching)
		requireMarker(t, evt, err, "marker", "matched")

		other := contractEvent()
		other.Fields["flag"] = "no"
		evt, err = procs.Run(other)
		require.NoError(t, err)
		require.NotNil(t, evt)
		_, gerr := evt.GetValue("marker")
		assert.Error(t, gerr, "processor must not run when the condition does not match")

		assert.NoError(t, procs.Close())
		assertAllClosed(t, id)
		assertLifecycleInvariants(t, id)
	})

	t.Run("drop is propagated", func(t *testing.T) {
		procs := buildContractProcessors(t, mapstr.M{"drop_event": mapstr.M{}})
		defer procs.Close()

		evt, err := procs.Run(contractEvent())
		assert.NoError(t, err)
		assert.Nil(t, evt, "dropped events must yield a nil event and nil error")
	})
}

func TestContractGroupCloseLifecycle(t *testing.T) {
	id := contractUniqueID("group-close")
	group := buildContractProcessors(t,
		closerCfg(id, "marker", "one"),
		closerCfg(id, "marker2", "two"),
	)

	evt, err := group.Run(contractEvent())
	requireMarker(t, evt, err, "marker", "one")
	requireMarker(t, evt, err, "marker2", "two")

	assert.NoError(t, group.Close())
	assertAllClosed(t, id)
	assertLifecycleInvariants(t, id)

	assert.NoError(t, group.Close())
	assertLifecycleInvariants(t, id)

	for _, p := range group.All() {
		assert.NoError(t, processors.Close(p))
	}
	assertLifecycleInvariants(t, id)
}

func TestContractRunAfterClose(t *testing.T) {
	id := contractUniqueID("run-after-close")
	group := buildContractProcessors(t,
		closerCfg(id, "marker", "wanted"),
		mapstr.M{"add_fields": mapstr.M{"target": "", "fields": mapstr.M{"extra": "yes"}}},
	)

	evt, err := group.Run(contractEvent())
	requireMarker(t, evt, err, "marker", "wanted")

	assert.NoError(t, group.Close())

	for range 3 {
		var rerr error
		var revt *beat.Event
		assert.NotPanics(t, func() {
			revt, rerr = group.Run(contractEvent())
		})
		assert.NoError(t, checkErrOrCorrect(revt, rerr, "marker", "wanted"))
	}
	assertLifecycleInvariants(t, id)
}

func TestContractSetPathsLifecycle(t *testing.T) {
	id := contractUniqueID("set-paths")
	group := buildContractProcessors(t, mapstr.M{
		contractPathSetterName: mapstr.M{"id": id, "field": "marker", "value": "with-paths"},
	})

	insts := contractInstances.get(id)
	require.NotEmpty(t, insts)

	// Pipelines discover PathSetter through group-entry type assertions.
	var setters []processors.PathSetter
	for _, p := range group.All() {
		if ps, ok := p.(processors.PathSetter); ok {
			setters = append(setters, ps)
		}
	}
	require.NotEmpty(t, setters, "a PathSetter processor must be reachable via the group entries")

	_, err := group.Run(contractEvent())
	assert.Error(t, err)
	for i, st := range insts {
		assert.Zerof(t, st.snapshot().runsBeforeSetPaths,
			"instance %d: underlying Run must not be called before SetPaths", i)
	}

	testPaths := &paths.Path{Home: "/contract/home-a"}
	for _, s := range setters {
		assert.NoError(t, s.SetPaths(testPaths))
	}
	for _, s := range setters {
		assert.NoError(t, s.SetPaths(testPaths))
	}
	for i, st := range insts {
		snap := st.snapshot()
		assert.LessOrEqualf(t, snap.setPathsCount, 1,
			"instance %d: underlying SetPaths must be called at most once", i)
		if snap.setPathsCount == 1 {
			assert.Equalf(t, "/contract/home-a", snap.pathsSeen[0].Home,
				"instance %d: underlying processor must receive the configured paths", i)
		}
	}

	conflicting := &paths.Path{Home: "/contract/home-b"}
	for _, s := range setters {
		assert.Error(t, s.SetPaths(conflicting))
	}
	for i, st := range insts {
		snap := st.snapshot()
		for _, seen := range snap.pathsSeen {
			assert.Equalf(t, "/contract/home-a", seen.Home,
				"instance %d: underlying processor must never observe conflicting paths", i)
		}
	}

	evt, err := group.Run(contractEvent())
	requireMarker(t, evt, err, "marker", "with-paths")

	assert.NoError(t, group.Close())
	assertAllClosed(t, id)

	for _, s := range setters {
		_ = s.SetPaths(testPaths) // returned error is allowed but not required
	}
	assertLifecycleInvariants(t, id)
}

func TestContractCrossPipelineIndependence(t *testing.T) {
	const numPipelines = 4

	cases := []struct {
		name         string
		sharedConfig bool
	}{
		{name: "identical config", sharedConfig: true},
		{name: "different configs", sharedConfig: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ids := make([]string, numPipelines)
			values := make([]string, numPipelines)
			sharedID := contractUniqueID("xpipe-shared")
			for i := range numPipelines {
				if tc.sharedConfig {
					ids[i] = sharedID
					values[i] = "shared-value"
				} else {
					ids[i] = contractUniqueID(fmt.Sprintf("xpipe-%d", i))
					values[i] = fmt.Sprintf("value-%d", i)
				}
			}

			pipelines := make([]*processors.Processors, numPipelines)
			for i := range pipelines {
				pipelines[i] = buildContractProcessors(t,
					closerCfg(ids[i], "marker", values[i]),
					mapstr.M{"add_fields": mapstr.M{"target": "", "fields": mapstr.M{"static": "static-value"}}},
				)
			}

			for i, pl := range pipelines {
				evt, err := pl.Run(contractEvent())
				requireMarker(t, evt, err, "marker", values[i])
				requireMarker(t, evt, err, "static", "static-value")
			}

			for i := range pipelines {
				assert.NoError(t, pipelines[i].Close())

				for j := i + 1; j < numPipelines; j++ {
					for range 3 {
						evt, err := pipelines[j].Run(contractEvent())
						requireMarker(t, evt, err, "marker", values[j])
						requireMarker(t, evt, err, "static", "static-value")
					}
				}

				evt, err := pipelines[i].Run(contractEvent())
				assert.NoError(t, checkErrOrCorrect(evt, err, "marker", values[i]))
			}

			for _, id := range ids {
				assertAllClosed(t, id)
				assertLifecycleInvariants(t, id)
			}
		})
	}
}

func TestContractProcessorsSharedAcrossGroups(t *testing.T) {
	id := contractUniqueID("multi-group")
	inner := buildContractProcessors(t, closerCfg(id, "marker", "inner-value"))

	outer := processors.NewList(logptest.NewTestingLogger(t, ""))
	outer.AddProcessors(*inner)
	outer.AddProcessor(addfields.NewAddFields(mapstr.M{"outer_marker": "yes"}, false, true))

	evt, err := outer.Run(contractEvent())
	requireMarker(t, evt, err, "marker", "inner-value")
	requireMarker(t, evt, err, "outer_marker", "yes")

	assert.NoError(t, outer.Close())
	assert.NoError(t, inner.Close())

	assertAllClosed(t, id)
	assertLifecycleInvariants(t, id)

	for _, group := range []*processors.Processors{outer, inner} {
		revt, rerr := group.Run(contractEvent())
		assert.NoError(t, checkErrOrCorrect(revt, rerr, "marker", "inner-value"))
	}
}

func TestContractConcurrentPipelines(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	const (
		workers     = 12
		iterations  = 4
		eventsPerIt = 8
	)

	sharedID := contractUniqueID("conc-shared")
	const sharedValue = "conc-shared-value"

	ids := []string{sharedID}
	workerID := make([]string, workers)
	workerValue := make([]string, workers)
	for w := range workers {
		if w%2 == 0 {
			workerID[w], workerValue[w] = sharedID, sharedValue
		} else {
			workerID[w] = contractUniqueID(fmt.Sprintf("conc-worker-%d", w))
			workerValue[w] = fmt.Sprintf("conc-value-%d", w)
			ids = append(ids, workerID[w])
		}
	}

	var wg sync.WaitGroup
	for w := range workers {
		id, value := workerID[w], workerValue[w]
		wg.Go(func() {
			for range iterations {
				if _, err := processors.GetConstructor(contractCloserName); err != nil {
					t.Errorf("worker %d: GetConstructor: %v", w, err)
					return
				}

				failCfg, err := processors.NewPluginConfigFromList([]mapstr.M{
					{contractFailingName: mapstr.M{}},
				})
				if err != nil {
					t.Errorf("worker %d: building failing config: %v", w, err)
					return
				}
				if _, err := processors.New(failCfg, logger); err == nil {
					t.Errorf("worker %d: expected constructor error, got none", w)
				}

				cfg, err := processors.NewPluginConfigFromList([]mapstr.M{
					closerCfg(id, "marker", value),
					{"add_fields": mapstr.M{"target": "", "fields": mapstr.M{"static": "static-value"}}},
				})
				if err != nil {
					t.Errorf("worker %d: building config: %v", w, err)
					return
				}
				group, err := processors.New(cfg, logger)
				if err != nil {
					t.Errorf("worker %d: processors.New: %v", w, err)
					return
				}

				for range eventsPerIt {
					evt, err := group.Run(contractEvent())
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
						t.Errorf("worker %d: marker = %v, want %q (event processed with another pipeline's config)",
							w, got, value)
					}
				}

				if err := group.Close(); err != nil {
					t.Errorf("worker %d: Close: %v", w, err)
				}
				if err := group.Close(); err != nil {
					t.Errorf("worker %d: double Close: %v", w, err)
				}
				evt, rerr := group.Run(contractEvent())
				if err := checkErrOrCorrect(evt, rerr, "marker", value); err != nil {
					t.Errorf("worker %d: Run after Close: %v", w, err)
				}
			}
		})
	}
	wg.Wait()

	for _, id := range ids {
		assertAllClosed(t, id)
		assertLifecycleInvariants(t, id)
	}
}

func TestContractConcurrentRunAndClose(t *testing.T) {
	id := contractUniqueID("run-vs-close")
	const value = "run-vs-close-value"
	group := buildContractProcessors(t, closerCfg(id, "marker", value))

	evt, err := group.Run(contractEvent())
	requireMarker(t, evt, err, "marker", value)

	const (
		runners    = 8
		preRuns    = 20
		racingRuns = 50
		postRuns   = 10
	)

	var preClose sync.WaitGroup
	preClose.Add(runners)
	closed := make(chan struct{})

	var wg sync.WaitGroup
	for r := range runners {
		wg.Go(func() {
			// Phase 1: strictly before Close, every run must succeed.
			for range preRuns {
				evt, err := group.Run(contractEvent())
				if err != nil {
					t.Errorf("runner %d: unexpected error before Close: %v", r, err)
					continue
				}
				if cerr := checkErrOrCorrect(evt, err, "marker", value); cerr != nil {
					t.Errorf("runner %d: before Close: %v", r, cerr)
				}
			}
			preClose.Done()

			// Phase 2: racing with Close.
			for range racingRuns {
				evt, err := group.Run(contractEvent())
				if cerr := checkErrOrCorrect(evt, err, "marker", value); cerr != nil {
					t.Errorf("runner %d: racing with Close: %v", r, cerr)
				}
			}

			// Phase 3: strictly after Close.
			<-closed
			for range postRuns {
				evt, err := group.Run(contractEvent())
				if cerr := checkErrOrCorrect(evt, err, "marker", value); cerr != nil {
					t.Errorf("runner %d: after Close: %v", r, cerr)
				}
			}
		})
	}

	preClose.Wait()
	assert.NoError(t, group.Close())
	assert.NoError(t, group.Close())
	close(closed)
	wg.Wait()

	assertAllClosed(t, id)
	assertLifecycleInvariants(t, id)
}
