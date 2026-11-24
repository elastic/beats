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

package secretstores

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/mapstr"

	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/libbeat/beat"
)

var errCreate = errors.New("invalid header")

// innerFactory makes plugins with one job per endpoint publishing the
// header and endpoint settings. A "bad" header fails Create.
type innerFactory struct {
	created, closed atomic.Int64
}

func (f *innerFactory) factory() plugin.PluginFactory {
	return plugin.PluginFactory{
		Name: "inner",
		Make: func(_ string, cfg *conf.C, info beat.Info) (plugin.Plugin, error) {
			c := struct {
				Header    string `config:"header"`
				Endpoints int    `config:"endpoints"`
			}{Endpoints: 1}
			if err := cfg.Unpack(&c); err != nil {
				return plugin.Plugin{}, err
			}
			closer := func() error {
				f.closed.Add(1)
				return nil
			}
			if c.Header == "bad" {
				// Resources returned with the error must be closed too.
				return plugin.Plugin{DoClose: closer}, errCreate
			}
			f.created.Add(1)
			js := make([]jobs.Job, c.Endpoints)
			for i := range js {
				js[i] = func(event *beat.Event) ([]jobs.Job, error) {
					event.Fields = mapstr.M{"header": c.Header, "endpoint": i}
					return nil, nil
				}
			}
			return plugin.Plugin{Jobs: js, Endpoints: c.Endpoints, DoClose: closer, Logger: info.Logger}, nil
		},
	}
}

// testResolver sets the header and endpoints settings, or fails.
type testResolver struct {
	mtx       sync.Mutex
	header    string
	endpoints int
	err       error
	calls     int
}

func (r *testResolver) set(header string, endpoints int, err error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.header, r.endpoints, r.err = header, endpoints, err
}

func (r *testResolver) resolve(cfg *conf.C) (*conf.C, *Scrubber, error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.calls++
	if r.err != nil {
		return nil, nil, r.err
	}
	c := conf.NewConfig()
	if err := c.Merge(cfg); err != nil {
		return nil, nil, err
	}
	if err := c.SetString("header", -1, r.header); err != nil {
		return nil, nil, err
	}
	return c, nil, c.SetInt("endpoints", -1, int64(r.endpoints))
}

func (r *testResolver) callCount() int {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	return r.calls
}

func makeSecretPlugin(t *testing.T, r *testResolver, f *innerFactory) plugin.Plugin {
	t.Helper()
	logger := logptest.NewTestingLogger(t, "")
	cfg := newTestConfig(t, map[string]any{"header": "fake:token"})
	resolve := func(_ context.Context, cfg *conf.C, _ map[string]Store) (*conf.C, *Scrubber, error) {
		return r.resolve(cfg)
	}
	p := newSecretPlugin(resolve, nil, f.factory(), "inner", cfg, beat.Info{Logger: logger}, logger, NewRedactor()).plugin()
	t.Cleanup(func() { _ = p.Close() })
	return p
}

// runJob runs job and returns the fields it published.
func runJob(t *testing.T, job jobs.Job) (mapstr.M, error) {
	t.Helper()
	event := &beat.Event{}
	_, err := job(event)
	return event.Fields, err
}

func TestSecretPluginRecoversFromResolutionError(t *testing.T) {
	r := &testResolver{err: errFakeUnavailable}
	f := &innerFactory{}
	p := makeSecretPlugin(t, r, f)
	require.Len(t, p.Jobs, 1, "without an inner plugin there must be one job")

	_, err := runJob(t, p.Jobs[0])
	assert.ErrorIs(t, err, errFakeUnavailable, "the job must report the resolution error")

	r.set("v1", 1, nil)
	fields, err := runJob(t, p.Jobs[0])
	require.NoError(t, err, "the job must recover once the config resolves")
	assert.Equal(t, "v1", fields["header"], "the job must use the resolved value")
	assert.Equal(t, int64(1), f.created.Load(), "the inner plugin must be created once")
}

func TestSecretPluginRotation(t *testing.T) {
	r := &testResolver{header: "v1", endpoints: 1}
	f := &innerFactory{}
	p := makeSecretPlugin(t, r, f)

	fields, err := runJob(t, p.Jobs[0])
	require.NoError(t, err, "the job must run")
	assert.Equal(t, "v1", fields["header"], "the job must use the resolved value")
	assert.Equal(t, int64(1), f.created.Load(), "an unchanged value must not create the inner plugin again")

	r.set("v2", 1, nil)
	fields, err = runJob(t, p.Jobs[0])
	require.NoError(t, err, "the job must run")
	assert.Equal(t, "v2", fields["header"], "the job must use the rotated value")
	assert.Equal(t, int64(2), f.created.Load(), "a rotated value must create the inner plugin again")
	assert.Equal(t, int64(1), f.closed.Load(), "the previous inner plugin must be closed")
}

func TestSecretPluginCreatesOncePerChange(t *testing.T) {
	r := &testResolver{header: "v1", endpoints: 8}
	f := &innerFactory{}
	p := makeSecretPlugin(t, r, f)
	require.Len(t, p.Jobs, 8, "there must be one job per endpoint")

	r.set("v2", 8, nil)
	var wg sync.WaitGroup
	for i, job := range p.Jobs {
		wg.Go(func() {
			fields, err := runJob(t, job)
			assert.NoError(t, err, "job %d must run", i)
			assert.Equal(t, i, fields["endpoint"], "job %d must run its endpoint", i)
		})
	}
	wg.Wait()
	assert.Equal(t, int64(2), f.created.Load(), "concurrent jobs must create the inner plugin once per change")
}

func TestSecretPluginCreateError(t *testing.T) {
	r := &testResolver{header: "v1", endpoints: 1}
	f := &innerFactory{}
	p := makeSecretPlugin(t, r, f)

	r.set("bad", 1, nil)
	_, err := runJob(t, p.Jobs[0])
	assert.ErrorIs(t, err, errCreate, "the job must report the create error")
	assert.Equal(t, int64(2), f.closed.Load(), "the previous plugin and the failed one must be closed")

	_, err = runJob(t, p.Jobs[0])
	assert.ErrorIs(t, err, errCreate, "the job must keep reporting the create error")
	assert.Equal(t, int64(2), f.closed.Load(), "the same config must not be created again")

	r.set("v2", 1, nil)
	fields, err := runJob(t, p.Jobs[0])
	require.NoError(t, err, "the job must recover once the value changes")
	assert.Equal(t, "v2", fields["header"], "the job must use the new value")
}

func TestSecretPluginEndpointCountChange(t *testing.T) {
	tests := map[string]struct {
		startErr  error
		start     int
		endpoints int
	}{
		"more endpoints after recovery": {startErr: errFakeUnavailable, endpoints: 2},
		"fewer endpoints":               {start: 2, endpoints: 1},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := &testResolver{header: "v1", endpoints: tc.start, err: tc.startErr}
			p := makeSecretPlugin(t, r, &innerFactory{})

			r.set("v1", tc.endpoints, nil)
			for i, job := range p.Jobs {
				_, err := runJob(t, job)
				assert.ErrorContains(t, err, "reconfigure the monitor", "job %d must ask to reconfigure the monitor", i)
			}
		})
	}
}

func TestSecretPluginUpdateDoesNotResolve(t *testing.T) {
	r := &testResolver{header: "v1", endpoints: 1}
	f := &innerFactory{}
	p := makeSecretPlugin(t, r, f)
	calls := r.callCount()

	r.set("", 0, errFakeUnavailable)
	updated := newTestConfig(t, map[string]any{"header": "fake:token", "other": "setting"})
	assert.NoError(t, p.Update(updated), "update must not fail when the store is unavailable")
	assert.Equal(t, calls, r.callCount(), "update must not resolve the config")

	r.set("v1", 1, nil)
	_, err := runJob(t, p.Jobs[0])
	require.NoError(t, err, "the job must run")
	assert.Equal(t, int64(2), f.created.Load(), "the next run must use the updated config")
}

func TestSecretPluginClose(t *testing.T) {
	r := &testResolver{header: "v1", endpoints: 1}
	f := &innerFactory{}
	p := makeSecretPlugin(t, r, f)

	require.NoError(t, p.Close(), "close must succeed")
	assert.Equal(t, int64(1), f.closed.Load(), "the inner plugin must be closed")

	r.set("v2", 1, nil)
	_, err := runJob(t, p.Jobs[0])
	assert.ErrorIs(t, err, errPluginClosed, "a closed plugin must not run")
	assert.Equal(t, int64(1), f.created.Load(), "a closed plugin must not create the inner plugin again")
}

// TestSecretPluginRedacts checks the plugin gives its redactor the secret
// values of each run, also when the plugin fails on them, and logs through
// it.
func TestSecretPluginRedacts(t *testing.T) {
	const secret = "s3cr3t-token"
	factory := plugin.PluginFactory{
		Name: "leaky",
		Make: func(_ string, cfg *conf.C, info beat.Info) (plugin.Plugin, error) {
			info.Logger.Infof("making with %s", secret)
			header, err := cfg.String("header", -1)
			if err != nil {
				return plugin.Plugin{}, err
			}
			if header == "invalid" {
				return plugin.Plugin{}, fmt.Errorf("invalid header %q", secret)
			}
			job := func(event *beat.Event) ([]jobs.Job, error) {
				event.Fields = mapstr.M{"url": mapstr.M{"full": "https://host/?t=" + secret}}
				return nil, nil
			}
			return plugin.Plugin{Jobs: []jobs.Job{job}, Endpoints: 1}, nil
		},
	}
	header := "valid"
	resolve := func(_ context.Context, cfg *conf.C, _ map[string]Store) (*conf.C, *Scrubber, error) {
		c := conf.NewConfig()
		if err := c.Merge(cfg); err != nil {
			return nil, nil, err
		}
		return c, scrubberFor(secret), c.SetString("header", -1, header)
	}
	redactor := NewRedactor()
	logger, logs := logptest.NewTestingLoggerWithObserver(t, "")
	p := newSecretPlugin(resolve, nil, factory, "leaky", newTestConfig(t, map[string]any{}), beat.Info{Logger: logger}, logger, redactor).plugin()
	t.Cleanup(func() { _ = p.Close() })

	event := &beat.Event{}
	_, err := p.Jobs[0](event)
	require.NoError(t, err, "the job must run")
	_, _ = redactor.Run(event)
	assert.Equal(t, mapstr.M{"url": mapstr.M{"full": "https://host/?t=<hidden>"}}, event.Fields, "the redactor must remove the values of the run")

	header = "invalid"
	_, err = p.Jobs[0](&beat.Event{})
	require.Error(t, err, "the creation error must be returned")
	assert.Equal(t, `invalid header "<hidden>"`, redactor.scrub(err.Error()), "the redactor must remove the values of a failed run")

	made := logs.FilterMessageSnippet("making with").TakeAll()
	require.NotEmpty(t, made, "the plugin must log with the logger it is given")
	for _, e := range made {
		assert.Equal(t, "making with <hidden>", e.Message, "the plugin logger must not write the secret")
	}
}

// TestSecretPluginCloseErrorAfterRotation checks closing the plugin of
// rotated values, which the redactor no longer has, doesn't log them.
func TestSecretPluginCloseErrorAfterRotation(t *testing.T) {
	factory := plugin.PluginFactory{
		Name: "closing",
		Make: func(_ string, cfg *conf.C, _ beat.Info) (plugin.Plugin, error) {
			header, err := cfg.String("header", -1)
			if err != nil {
				return plugin.Plugin{}, err
			}
			job := func(*beat.Event) ([]jobs.Job, error) { return nil, nil }
			closer := func() error { return fmt.Errorf("could not close %s", header) }
			return plugin.Plugin{Jobs: []jobs.Job{job}, Endpoints: 1, DoClose: closer}, nil
		},
	}
	header := "first-secret"
	resolve := func(_ context.Context, cfg *conf.C, _ map[string]Store) (*conf.C, *Scrubber, error) {
		c := conf.NewConfig()
		if err := c.Merge(cfg); err != nil {
			return nil, nil, err
		}
		return c, scrubberFor(header), c.SetString("header", -1, header)
	}
	logger, logs := logptest.NewTestingLoggerWithObserver(t, "")
	p := newSecretPlugin(resolve, nil, factory, "closing", newTestConfig(t, map[string]any{}), beat.Info{Logger: logger}, logger, NewRedactor()).plugin()
	t.Cleanup(func() { _ = p.Close() })

	header = "second-secret"
	_, err := p.Jobs[0](&beat.Event{})
	require.NoError(t, err, "the job must run with the rotated value")

	closed := logs.FilterMessageSnippet("Could not close").TakeAll()
	require.Len(t, closed, 1, "closing the plugin of the first value must be logged")
	assert.Equal(t, "Could not close the monitor plugin: could not close <hidden>", closed[0].Message, "the close error must not have the rotated value")
}

func TestWrapFactoryKeepsFactory(t *testing.T) {
	f := (&innerFactory{}).factory()
	f.Aliases = []string{"alias"}
	wrapped := NewResolver(logptest.NewTestingLogger(t, "")).WrapFactory(f, NewRedactor())
	assert.Equal(t, f.Name, wrapped.Name, "the name must be kept")
	assert.Equal(t, f.Aliases, wrapped.Aliases, "the aliases must be kept")
}

func TestWrapFactoryEviction(t *testing.T) {
	resetSharedStores(t)
	logger := logptest.NewTestingLogger(t, "")
	wrapped := NewResolver(logger).WrapFactory((&innerFactory{}).factory(), NewRedactor())

	makeCfg := func(header string, stores ...any) *conf.C {
		m := map[string]any{"type": "http", "header": header}
		if len(stores) > 0 {
			m[ConfigKey] = stores
		}
		return newTestConfig(t, m)
	}
	conn := func(name, id string) map[string]any {
		return map[string]any{"type": fakeType, "name": name, "id": id}
	}

	// Two monitors sharing one connection (plus a duplicate identity in p1) pin a single store.
	p1, err := wrapped.Make("inner", makeCfg("fake:a@tok", conn("a", "shared"), conn("dup", "shared")), beat.Info{Logger: logger})
	require.NoError(t, err, "first plugin must be created")
	p2, err := wrapped.Make("inner", makeCfg("fake:b@tok", conn("b", "shared")), beat.Info{Logger: logger})
	require.NoError(t, err, "second plugin must be created")
	assert.Equal(t, 1, sharedStoreCount(), "equal connections must share one store")

	require.NoError(t, p1.Close(), "closing first plugin must succeed")
	require.NoError(t, p1.Close(), "closing twice must be idempotent")
	assert.Equal(t, 1, sharedStoreCount(), "store must stay cached while second monitor holds it")
	assert.Equal(t, int64(0), fakeStoreCloses.Load(), "store must stay open while referenced")

	fields, err := runJob(t, p2.Jobs[0])
	require.NoError(t, err, "second monitor must keep resolving")
	assert.Equal(t, "shared-tok", fields["header"], "second monitor must use the shared store")

	// Updating p2 to add connB keeps shared alive; dropping shared evicts only shared.
	require.NoError(t, p2.Update(makeCfg("fake:b@tok", conn("b", "shared"), conn("c", "connB"))), "update adding connB must succeed")
	assert.Equal(t, 2, sharedStoreCount(), "both connections must be cached")
	assert.Equal(t, int64(0), fakeStoreCloses.Load(), "kept connection must not be closed")

	require.NoError(t, p2.Update(makeCfg("fake:c@tok", conn("c", "connB"))), "update removing shared must succeed")
	assert.Equal(t, 1, sharedStoreCount(), "removed connection must be evicted")
	assert.Equal(t, int64(1), fakeStoreCloses.Load(), "evicted store must be closed")

	// Updating to an invalid secret_stores rolls back newly acquired entries and evicts old ones.
	require.NoError(t, p2.Update(makeCfg("fake:a@tok", conn("a", "connA"), map[string]any{"type": fakeType, "name": "bad", "invalid": true})), "invalid update must not fail Update")
	assert.Zero(t, sharedStoreCount(), "invalid update must evict old stores without leaking new ones")
	_, err = runJob(t, p2.Jobs[0])
	assert.ErrorContains(t, err, "invalid fake config", "job must report the invalid store config")

	require.NoError(t, p2.Close(), "closing second plugin must succeed")
	assert.Zero(t, sharedStoreCount(), "no stores must remain cached")
}

const blockingType = "fake_blocking"

var lastBlockingStore atomic.Pointer[blockingStore]

func init() {
	mustRegister(blockingType, func(*conf.C, *logp.Logger) (Store, error) {
		bs := &blockingStore{started: make(chan struct{}, 1)}
		lastBlockingStore.Store(bs)
		return bs, nil
	})
}

type blockingStore struct {
	blockNext atomic.Bool
	started   chan struct{}
	closed    atomic.Int64
}

func (s *blockingStore) Validate(*conf.C, Scope) error { return nil }

func (s *blockingStore) Replace(ctx context.Context, cfg *conf.C, _ Scope) (*conf.C, error) {
	if !s.blockNext.Load() {
		return cfg, nil
	}
	select {
	case s.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func (s *blockingStore) Close() error {
	s.closed.Add(1)
	return nil
}

func TestWrapFactoryCloseCancelsInFlightResolve(t *testing.T) {
	resetSharedStores(t)
	logger := logptest.NewTestingLogger(t, "")
	f := &innerFactory{}
	wrapped := NewResolver(logger).WrapFactory(f.factory(), NewRedactor())

	cfg := newTestConfig(t, map[string]any{
		"type":   "http",
		"header": "plain",
		ConfigKey: []any{
			map[string]any{"type": blockingType, "name": "main"},
		},
	})

	p, err := wrapped.Make("inner", cfg, beat.Info{Logger: logger})
	require.NoError(t, err, "plugin must be created")
	bs := lastBlockingStore.Load()
	require.NotNil(t, bs, "blocking store must be created")

	bs.blockNext.Store(true)
	jobErr := make(chan error, 1)
	go func() {
		_, err := runJob(t, p.Jobs[0])
		jobErr <- err
	}()

	select {
	case <-bs.started:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "timed out waiting for in-flight Replace to start")
	}

	require.NoError(t, p.Close(), "closing plugin must succeed")
	select {
	case err := <-jobErr:
		assert.ErrorIs(t, err, context.Canceled, "closing plugin must cancel in-flight Replace")
	case <-time.After(5 * time.Second):
		require.FailNow(t, "timed out waiting for in-flight Replace to abort on close")
	}
	assert.Zero(t, sharedStoreCount(), "store must stay evicted after canceled in-flight Replace")
	assert.Equal(t, int64(1), bs.closed.Load(), "store must be closed once")
}
