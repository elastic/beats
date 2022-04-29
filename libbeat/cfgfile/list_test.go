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

package cfgfile

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
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

type runnerFactory struct {
	CreateRunner func(beat.PipelineConnector, *common.Config) (Runner, error)
	runners      []Runner
}

func (r *runnerFactory) Create(x beat.PipelineConnector, c *common.Config) (Runner, error) {
	config := struct {
		ID int64 `config:"id"`
	}{}

	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	// id < 0 is an invalid config
	if config.ID < 0 {
		return nil, errors.New("Invalid config")
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

func (r *runnerFactory) CheckConfig(config *common.Config) error {
	return nil
}

func TestNewConfigs(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)

	list.Reload([]*reload.ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	assert.Equal(t, len(list.copyRunnerList()), 3)
}

func TestReloadSameConfigs(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)

	list.Reload([]*reload.ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	state := list.copyRunnerList()
	assert.Equal(t, len(state), 3)

	list.Reload([]*reload.ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	// nothing changed
	assert.Equal(t, state, list.copyRunnerList())
}

func TestReloadDuplicateConfig(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)

	list.Reload([]*reload.ConfigWithMeta{
		createConfig(1),
	})

	state := list.copyRunnerList()
	assert.Equal(t, len(state), 1)

	// This can happen in Autodiscover when a container if getting restarted
	// but the previous one is not cleaned yet.
	list.Reload([]*reload.ConfigWithMeta{
		createConfig(1),
		createConfig(1),
	})

	// nothing changed
	assert.Equal(t, state, list.copyRunnerList())
}

func TestReloadStopConfigs(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)

	list.Reload([]*reload.ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	assert.Equal(t, len(list.copyRunnerList()), 3)

	list.Reload([]*reload.ConfigWithMeta{
		createConfig(1),
		createConfig(3),
	})

	assert.Equal(t, len(list.copyRunnerList()), 2)
}

func TestReloadStartStopConfigs(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)

	list.Reload([]*reload.ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	state := list.copyRunnerList()
	assert.Equal(t, len(state), 3)

	list.Reload([]*reload.ConfigWithMeta{
		createConfig(1),
		createConfig(3),
		createConfig(4),
	})

	assert.Equal(t, len(list.copyRunnerList()), 3)
	assert.NotEqual(t, state, list.copyRunnerList())
}

func TestStopAll(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)

	list.Reload([]*reload.ConfigWithMeta{
		createConfig(1),
		createConfig(2),
		createConfig(3),
	})

	assert.Equal(t, len(list.copyRunnerList()), 3)
	list.Stop()
	assert.Equal(t, len(list.copyRunnerList()), 0)

	for _, r := range list.runners {
		assert.False(t, r.(*runner).stopped)
	}
}

func TestHas(t *testing.T) {
	factory := &runnerFactory{}
	list := NewRunnerList("", factory, nil)
	config := createConfig(1)

	hash, err := HashConfig(config.Config)
	if err != nil {
		t.Fatal(err)
	}

	list.Reload([]*reload.ConfigWithMeta{
		config,
	})

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
				CreateRunner: func(p beat.PipelineConnector, cfg *common.Config) (Runner, error) {
					return &runner{
						OnStart: func() {
							c, _ := p.Connect()
							c.Close()
						},
					}, nil
				},
			}

			var config beat.ClientConfig
			pipeline := &pubtest.FakeConnector{
				ConnectFunc: func(cfg beat.ClientConfig) (beat.Client, error) {
					config = cfg
					return &pubtest.FakeClient{}, nil
				},
			}

			runner, _ := createRunner(factory, pipeline, &reload.ConfigWithMeta{
				Config: common.NewConfig(),
				Meta:   test.meta,
			})
			runner.Start()
			runner.Stop()

			assert.Equal(t, test.meta, config.Processing.DynamicFields)
		})
	}
}

func createConfig(id int64) *reload.ConfigWithMeta {
	c := common.NewConfig()
	c.SetInt("id", -1, id)
	return &reload.ConfigWithMeta{
		Config: c,
	}
}
