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

	for _, e := range events {
		c.publishes = append(c.publishes, e)
	}
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

func mockEventMonitorValidator(id string) validator.Validator {
	var idMatcher isdef.IsDef
	if id == "" {
		idMatcher = isdef.IsStringMatching(regexp.MustCompile(`^auto-test-.*`))
	} else {
		idMatcher = isdef.IsEqual(id)
	}
	return lookslike.Strict(lookslike.Compose(
		lookslike.MustCompile(map[string]interface{}{
			"monitor": map[string]interface{}{
				"id":          idMatcher,
				"name":        "",
				"type":        "test",
				"duration.us": isdef.IsDuration,
				"status":      "up",
				"check_group": isdef.IsString,
			},
		}),
		hbtestllext.MonitorTimespanValidator,
		hbtest.SummaryChecks(1, 0),
		lookslike.MustCompile(mockEventCustomFields()),
	))
}

func mockEventCustomFields() map[string]interface{} {
	return common.MapStr{"foo": "bar"}
}

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
			Builder: func(s string, config *common.Config) (plugin.Plugin, error) {
				built.Inc()
				// Declare a real config block with a required attr so we can see what happens when it doesn't work
				unpacked := struct {
					URLs []string `config:"urls" validate:"required"`
				}{}
				err := config.Unpack(&unpacked)
				if err != nil {
					return plugin.Plugin{}, err
				}
				j, err := createMockJob()
				closer := func() error {
					closed.Inc()
					return nil
				}
				return plugin.Plugin{Jobs: j, Close: closer, Endpoints: 1}, err
			},
			Stats: plugin.NewPluginCountersRecorder("test", reg)},
		built,
		closed
}

func mockPluginsReg() (p *plugin.PluginsReg, built *atomic.Int, closed *atomic.Int) {
	reg := plugin.NewPluginsReg()
	builder, built, closed := mockPluginBuilder()
	reg.Add(builder)
	return reg, built, closed
}

func mockPluginConf(t *testing.T, id string, schedule string, url string) *common.Config {
	confMap := map[string]interface{}{
		"type":     "test",
		"urls":     []string{url},
		"schedule": schedule,
	}

	if id != "" {
		confMap["id"] = id
	}

	conf, err := common.NewConfigFrom(confMap)
	require.NoError(t, err)

	return conf
}

// mockBadPluginConf returns a conf with an invalid plugin config.
// This should fail after the generic plugin checks fail since the HTTP plugin requires 'urls' to be set.
func mockBadPluginConf(t *testing.T, id string, schedule string) *common.Config {
	confMap := map[string]interface{}{
		"type":        "test",
		"notanoption": []string{"foo"},
		"schedule":    schedule,
	}

	if id != "" {
		confMap["id"] = id
	}

	conf, err := common.NewConfigFrom(confMap)
	require.NoError(t, err)

	return conf
}

// mockInvalidPlugin conf returns a config that invalid at the basic level of
// what's expected in heartbeat, i.e. no type.
func mockInvalidPluginConf(t *testing.T) *common.Config {
	confMap := map[string]interface{}{
		"hoeutnheou": "oueanthoue",
	}

	conf, err := common.NewConfigFrom(confMap)
	require.NoError(t, err)

	return conf
}
