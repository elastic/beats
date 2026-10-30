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
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"

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

func (r *testResolver) resolve(cfg *conf.C) (*conf.C, error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	c := conf.NewConfig()
	if err := c.Merge(cfg); err != nil {
		return nil, err
	}
	if err := c.SetString("header", -1, r.header); err != nil {
		return nil, err
	}
	return c, c.SetInt("endpoints", -1, int64(r.endpoints))
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
	p := newSecretPlugin(r.resolve, f.factory(), "inner", cfg, beat.Info{Logger: logger}, logger).plugin()
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

func TestWrapFactoryKeepsFactory(t *testing.T) {
	f := (&innerFactory{}).factory()
	f.Aliases = []string{"alias"}
	wrapped := NewResolver(logptest.NewTestingLogger(t, "")).WrapFactory(f)
	assert.Equal(t, f.Name, wrapped.Name, "the name must be kept")
	assert.Equal(t, f.Aliases, wrapped.Aliases, "the aliases must be kept")
}
