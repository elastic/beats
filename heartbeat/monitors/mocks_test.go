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

package monitors

import (
	"fmt"
	"regexp"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/hbtest"
	"github.com/elastic/beats/v7/heartbeat/hbtestllext"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/atomic"
	"github.com/elastic/beats/v7/libbeat/monitoring"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/go-lookslike"
	"github.com/elastic/go-lookslike/isdef"
	"github.com/elastic/go-lookslike/validator"
)

type MockBeatClient struct {
	publishes []beat.Event
	closed    bool
	mtx       sync.Mutex
}

func (c *MockBeatClient) Publish(e beat.Event) {
	c.PublishAll([]beat.Event{e})
}

func (c *MockBeatClient) PublishAll(events []beat.Event) {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.publishes = append(c.publishes, events...)
}

func (c *MockBeatClient) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.closed {
		return fmt.Errorf("mock client already closed")
	}

	c.closed = true
	return nil
}

func (c *MockBeatClient) Publishes() []beat.Event {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	dst := make([]beat.Event, len(c.publishes))
	copy(dst, c.publishes)
	return dst
}

type MockPipelineConnector struct {
	clients []*MockBeatClient
	mtx     sync.Mutex
}

func (pc *MockPipelineConnector) Connect() (beat.Client, error) {
	return pc.ConnectWith(beat.ClientConfig{})
}

func (pc *MockPipelineConnector) ConnectWith(beat.ClientConfig) (beat.Client, error) {
	pc.mtx.Lock()
	defer pc.mtx.Unlock()

	c := &MockBeatClient{}

	pc.clients = append(pc.clients, c)

	return c, nil
}

func baseMockEventMonitorValidator(id string, name string, status string) validator.Validator {
	var idMatcher isdef.IsDef
	if id == "" {
		idMatcher = isdef.IsStringMatching(regexp.MustCompile(`^auto-test-.*`))
	} else {
		idMatcher = isdef.IsEqual(id)
	}
	return lookslike.MustCompile(map[string]interface{}{
		"monitor": map[string]interface{}{
			"id":          idMatcher,
			"name":        name,
			"type":        "test",
			"duration.us": hbtestllext.IsInt64,
			"status":      status,
			"check_group": isdef.IsString,
		},
	})
}

func mockEventMonitorValidator(id string, name string) validator.Validator {
	return lookslike.Strict(lookslike.Compose(
		baseMockEventMonitorValidator(id, name, "up"),
		hbtestllext.MonitorTimespanValidator,
		hbtest.SummaryChecks(1, 0),
		lookslike.MustCompile(mockEventCustomFields()),
	))
}

func mockEventCustomFields() map[string]interface{} {
	return common.MapStr{"foo": "bar"}
}

//nolint:unparam // There are no new changes to this line but
// linter has been activated in the meantime. We'll cleanup separately.
func createMockJob() ([]jobs.Job, error) {
	j := jobs.MakeSimpleJob(func(event *beat.Event) error {
		eventext.MergeEventFields(event, mockEventCustomFields())
		return nil
	})

	return []jobs.Job{j}, nil
}

func mockPluginBuilder() (plugin.PluginFactory, *atomic.Int, *atomic.Int) {
	reg := monitoring.NewRegistry()

	built := atomic.NewInt(0)
	closed := atomic.NewInt(0)

	return plugin.PluginFactory{
			Name:    "test",
			Aliases: []string{"testAlias"},
			Make: func(s string, config *conf.C) (plugin.Plugin, error) {
				built.Inc()
				// Declare a real config block with a required attr so we can see what happens when it doesn't work
				unpacked := struct {
					URLs []string `config:"urls" validate:"required"`
				}{}

				// track all closes, even on error
				closer := func() error {
					closed.Inc()
					return nil
				}

				err := config.Unpack(&unpacked)
				if err != nil {
					return plugin.Plugin{DoClose: closer}, err
				}
				j, err := createMockJob()

				return plugin.Plugin{Jobs: j, DoClose: closer, Endpoints: 1}, err
			},
			Stats: plugin.NewPluginCountersRecorder("test", reg)},
		built,
		closed
}

func mockPluginsReg() (p *plugin.PluginsReg, built *atomic.Int, closed *atomic.Int) {
	reg := plugin.NewPluginsReg()
	builder, built, closed := mockPluginBuilder()
	//nolint:errcheck // There are no new changes to this line but
	// linter has been activated in the meantime. We'll cleanup separately.
	reg.Add(builder)
	return reg, built, closed
}

func mockPluginConf(t *testing.T, id string, name string, schedule string, url string) *conf.C {
	confMap := map[string]interface{}{
		"type":     "test",
		"urls":     []string{url},
		"schedule": schedule,
		"name":     name,
	}

	// Optional to let us simulate this key missing
	if id != "" {
		confMap["id"] = id
	}

	conf, err := conf.NewConfigFrom(confMap)
	require.NoError(t, err)

	return conf
}

// mockBadPluginConf returns a conf with an invalid plugin config.
// This should fail after the generic plugin checks fail since the HTTP plugin requires 'urls' to be set.
//nolint:unparam // There are no new changes to this line but
// linter has been activated in the meantime. We'll cleanup separately.
func mockBadPluginConf(t *testing.T, id string, schedule string) *common.Config {
	confMap := map[string]interface{}{
		"type":        "test",
		"notanoption": []string{"foo"},
	}

	if id != "" {
		confMap["id"] = id
	}

	conf, err := conf.NewConfigFrom(confMap)
	require.NoError(t, err)

	return conf
}

func mockInvalidPluginConf(t *testing.T) *conf.C {
	confMap := map[string]interface{}{
		"hoeutnheou": "oueanthoue",
	}

	conf, err := conf.NewConfigFrom(confMap)
	require.NoError(t, err)

	return conf
}

func mockInvalidPluginConfWithStdFields(t *testing.T, id string, name string, schedule string) *conf.C {
	confMap := map[string]interface{}{
		"type":     "test",
		"id":       id,
		"name":     name,
		"schedule": schedule,
	}

	conf, err := conf.NewConfigFrom(confMap)
	require.NoError(t, err)

	return conf
}
