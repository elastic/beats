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

	"github.com/elastic/beats/heartbeat/hbtest"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/heartbeat/eventext"
	"github.com/elastic/beats/heartbeat/monitors/jobs"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/mapval"
	"github.com/elastic/beats/libbeat/monitoring"
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

func mockEventMonitorValidator(id string) mapval.Validator {
	var idMatcher mapval.IsDef
	if id == "" {
		idMatcher = mapval.IsStringMatching(regexp.MustCompile(`^auto-test-.*`))
	} else {
		idMatcher = mapval.IsEqual(id)
	}
	return mapval.Strict(mapval.Compose(
		mapval.MustCompile(mapval.Map{
			"monitor": mapval.Map{
				"id":          idMatcher,
				"name":        "",
				"type":        "test",
				"duration.us": mapval.IsDuration,
				"status":      "up",
				"check_group": mapval.IsString,
			},
		}),
		hbtest.SummaryChecks(1, 0),
		mapval.MustCompile(mockEventCustomFields()),
	))
}

func mockEventCustomFields() map[string]interface{} {
	return common.MapStr{"foo": "bar"}
}

func createMockJob(name string, cfg *common.Config) ([]jobs.Job, error) {
	j := jobs.MakeSimpleJob(func(event *beat.Event) error {
		eventext.MergeEventFields(event, mockEventCustomFields())
		return nil
	})

	return []jobs.Job{j}, nil
}

func mockPluginBuilder() pluginBuilder {
	reg := monitoring.NewRegistry()

	return pluginBuilder{"test", ActiveMonitor, func(s string, config *common.Config) ([]jobs.Job, int, error) {
		c := common.Config{}
		j, err := createMockJob("test", &c)
		return j, 1, err
	}, newPluginCountersRecorder("test", reg)}
}

func mockPluginsReg() *pluginsReg {
	reg := newPluginsReg()
	reg.add(mockPluginBuilder())
	return reg
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
