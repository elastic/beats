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

package reload

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/diagnostics"
	rl "github.com/elastic/beats/v7/libbeat/common/reload"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type runner struct {
	id      int64
	started bool
	stopped bool
	OnStart func()
	OnStop  func()
}

func (r *runner) String() string {
	return "test runner"
}

func (r *runner) Start() {
	r.started = true
	if r.OnStart != nil {
		r.OnStart()
	}
}

func (r *runner) Stop() {
	if r.OnStop != nil {
		r.OnStop()
	}
	r.stopped = true
}

func (r *runner) Diagnostics() []diagnostics.DiagnosticSetup {
	return []diagnostics.DiagnosticSetup{
		{
			Name:     "test-callback",
			Callback: func() []byte { return []byte("test") },
		},
	}
}

type runnerFactory struct {
	CreateRunner func(beat.PipelineConnector, *config.C) (cfgfile.Runner, error)
	runners      []cfgfile.Runner
}

func (r *runnerFactory) Create(x beat.PipelineConnector, c *config.C) (cfgfile.Runner, error) {
	config := struct {
		ID int64 `config:"id"`
	}{}

	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	// id < 0 is an invalid config
	if config.ID < 0 {
		return nil, fmt.Errorf("Invalid config")
	}

	if r.CreateRunner != nil {
		runner, err := r.CreateRunner(x, c)
		if err != nil {
			return nil, err
		}
		r.runners = append(r.runners, runner)
		return runner, err
	}

	runner := &runner{id: config.ID}
	r.runners = append(r.runners, runner)
	return runner, err
}

func (r *runnerFactory) CheckConfig(_ *config.C) error {
	return nil
}

type testDiagHandler struct {
	gotResp string
}

func (r *testDiagHandler) Register(_ string, _ string, _ string, _ string, callback func() []byte) {
	r.gotResp = string(callback())
}

func TestDiagnostics(t *testing.T) {
	factory := &runnerFactory{}
	logger := logptest.NewTestingLogger(t, "")
	list := NewHBRunnerList("", factory, nil, logger)
	cfg := createConfig(1)
	callback := &testDiagHandler{}
	cfg.DiagCallback = callback
	err := list.Reload([]*rl.ConfigWithMeta{
		cfg,
	})

	require.NoError(t, err)
	require.Equal(t, "test", callback.gotResp)
}

func TestNewConfigs(t *testing.T) {
	factory := &runnerFactory{}
	logger := logptest.NewTestingLogger(t, "")
	list := NewHBRunnerList("", factory, nil, logger)

	err := list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	require.NoError(t, err)
	assert.Len(t, list.copyRunnerList(), 3)
}

func TestReloadSameConfigs(t *testing.T) {
	factory := &runnerFactory{}
	logger := logptest.NewTestingLogger(t, "")

	list := NewHBRunnerList("", factory, nil, logger)

	err := list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})
	require.NoError(t, err)

	state := list.copyRunnerList()
	assert.Len(t, state, 3)

	err = list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	// nothing changed
	require.NoError(t, err)
	assert.Equal(t, state, list.copyRunnerList())
}

func TestReloadDuplicateConfig(t *testing.T) {
	factory := &runnerFactory{}
	logger := logptest.NewTestingLogger(t, "")
	list := NewHBRunnerList("", factory, nil, logger)

	err := list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
	})
	require.NoError(t, err)

	state := list.copyRunnerList()
	assert.Len(t, state, 1)

	// This can happen in Autodiscover when a container if getting restarted
	// but the previous one is not cleaned yet.
	err = list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
		createConfig(1),
	})

	// nothing changed
	require.NoError(t, err)
	assert.Equal(t, state, list.copyRunnerList())
}

func TestReloadStopConfigs(t *testing.T) {
	factory := &runnerFactory{}
	logger := logptest.NewTestingLogger(t, "")
	list := NewHBRunnerList("", factory, nil, logger)

	err := list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	require.NoError(t, err)
	assert.Len(t, list.copyRunnerList(), 3)

	err = list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
		createConfig(3),
	})

	require.NoError(t, err)
	assert.Len(t, list.copyRunnerList(), 2)
}

func TestReloadStartStopConfigs(t *testing.T) {
	factory := &runnerFactory{}
	logger := logptest.NewTestingLogger(t, "")

	list := NewHBRunnerList("", factory, nil, logger)

	err := list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})
	require.NoError(t, err)

	state := list.copyRunnerList()
	assert.Len(t, state, 3)

	err = list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
		createConfig(3),
		createConfig(4),
	})

	require.NoError(t, err)
	assert.Len(t, list.copyRunnerList(), 3)
	assert.NotEqual(t, state, list.copyRunnerList())
}

func TestStopAll(t *testing.T) {
	factory := &runnerFactory{}
	logger := logptest.NewTestingLogger(t, "")

	list := NewHBRunnerList("", factory, nil, logger)

	err := list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	require.NoError(t, err)
	assert.Len(t, list.copyRunnerList(), 3)

	runners := list.Runners()
	list.Stop()
	assert.Empty(t, list.copyRunnerList())

	for _, rr := range runners {
		r, ok := rr.(*runner)
		assert.True(t, ok)
		assert.True(t, r.stopped)
	}
}

func TestHas(t *testing.T) {
	factory := &runnerFactory{}
	logger := logptest.NewTestingLogger(t, "")

	list := NewHBRunnerList("", factory, nil, logger)
	config := createConfig(1)

	hash, err := list.HashConfig(config.Config)
	require.NoError(t, err)

	err = list.Reload([]*rl.ConfigWithMeta{
		config,
	})

	require.NoError(t, err)
	assert.True(t, list.Has(hash))
	assert.False(t, list.Has(0))
}

func TestCreateRunnerAddsDynamicMeta(t *testing.T) {
	newMapStrPointer := func(m mapstr.M) *mapstr.Pointer {
		p := mapstr.NewPointer(m)
		return &p
	}

	cases := map[string]struct {
		meta *mapstr.Pointer
	}{
		"no dynamic metadata": {},
		"with dynamic fields": {
			meta: newMapStrPointer(mapstr.M{"test": 1}),
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {

			factory := &runnerFactory{
				CreateRunner: func(p beat.PipelineConnector, cfg *config.C) (cfgfile.Runner, error) {
					return &runner{
						OnStart: func() {
							c, _ := p.Connect()
							c.Close()
						},
					}, nil
				},
			}

			var c beat.ClientConfig
			pipeline := &pubtest.FakeConnector{
				ConnectFunc: func(cfg beat.ClientConfig) (beat.Client, error) {
					c = cfg
					return &pubtest.FakeClient{}, nil
				},
			}

			runner, _ := createRunner(factory, pipeline, &rl.ConfigWithMeta{
				Config: config.NewConfig(),
				Meta:   test.meta,
			})
			runner.Start()
			runner.Stop()

			assert.Equal(t, test.meta, c.Processing.DynamicFields)
		})
	}
}

func createConfig(id int64) *rl.ConfigWithMeta {
	c := config.NewConfig()
	_ = c.SetInt("id", -1, id)
	return &rl.ConfigWithMeta{
		Config: c,
	}
}

// This section applies to heartbeat extension of libbbeat's list rler
// to add dynamic update and custom config hashing.
// Setting custom runners and factory

type updatableRunner struct {
	id       int64
	updated  bool
	OnStart  func()
	OnStop   func()
	OnUpdate func(c *config.C) error
}

func (r *updatableRunner) String() string {
	return "test runner"
}

func (r *updatableRunner) Start() {
	if r.OnStart != nil {
		r.OnStart()
	}
}

func (r *updatableRunner) Stop() {
	if r.OnStop != nil {
		r.OnStop()
	}
}

func (r *updatableRunner) Update(c *config.C) (err error) {
	if r.OnUpdate != nil {
		err = r.OnUpdate(c)
	}
	r.updated = true
	return err
}

var _ UpdatableRunner = &updatableRunner{}

type hbrunnerFactory struct {
	CreateRunner func(beat.PipelineConnector, *config.C) (cfgfile.Runner, error)
	runners      []cfgfile.Runner
	hashFunc     plugin.HashConfigFunc
}

func (r *hbrunnerFactory) Create(x beat.PipelineConnector, c *config.C) (cfgfile.Runner, error) {
	config := struct {
		ID int64 `config:"id"`
	}{}

	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	// id < 0 is an invalid config
	if config.ID < 0 {
		return nil, fmt.Errorf("Invalid config")
	}

	if r.CreateRunner != nil {
		runner, err := r.CreateRunner(x, c)
		if err != nil {
			return nil, err
		}
		r.runners = append(r.runners, runner)
		return runner, err
	}

	runner := &updatableRunner{id: config.ID}
	r.runners = append(r.runners, runner)
	return runner, err
}

func (r *hbrunnerFactory) CheckConfig(_ *config.C) error {
	return nil
}

func (r *hbrunnerFactory) GetHashFunc(c *config.C) (plugin.HashConfigFunc, error) {
	if r.hashFunc != nil {
		return r.hashFunc, nil
	}

	return nil, nil
}

var _ monitors.HBRunnerFactory = &hbrunnerFactory{}

func TestUpdateConfigs(t *testing.T) {
	factory := &hbrunnerFactory{}
	logger := logptest.NewTestingLogger(t, "")

	list := NewHBRunnerList("", factory, nil, logger)

	err := list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
	})
	require.NoError(t, err)

	state := list.copyRunnerList()
	assert.Len(t, state, 1)

	err = list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
	})

	require.NoError(t, err)
	assert.Equal(t, state, list.copyRunnerList())

	for _, runner := range list.copyRunnerList() {
		hbrunner, ok := runner.(*updatableRunner)
		require.True(t, ok)
		assert.True(t, hbrunner.updated)
	}
}

func TestUpdateConfigsError(t *testing.T) {
	factory := &hbrunnerFactory{CreateRunner: func(pc beat.PipelineConnector, c *config.C) (cfgfile.Runner, error) {
		cfg := struct {
			ID int64 `config:"id"`
		}{}
		err := c.Unpack(&cfg)
		if err != nil {
			return nil, err
		}

		return &updatableRunner{
			OnUpdate: func(c *config.C) error { return fmt.Errorf("test update config error") },
		}, nil
	}}
	logger := logptest.NewTestingLogger(t, "")

	list := NewHBRunnerList("", factory, nil, logger)

	err := list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
	})
	require.NoError(t, err)

	state := list.copyRunnerList()
	assert.Len(t, state, 1)

	err = list.Reload([]*rl.ConfigWithMeta{
		createConfig(1),
	})

	// nothing changed
	require.NoError(t, err)
	assert.Len(t, list.copyRunnerList(), 1)
	assert.NotEqual(t, state, list.copyRunnerList())

	for _, runner := range list.copyRunnerList() {
		hbrunner, ok := runner.(*updatableRunner)
		require.True(t, ok)
		assert.False(t, hbrunner.updated)
	}
}
