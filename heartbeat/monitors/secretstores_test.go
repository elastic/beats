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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
)

const unresolvedHeader = "Bearer ref://token"

var errResolve = errors.New("store unavailable")

// resolveHeader is a ConfigResolver that replaces the header setting.
func resolveHeader(header string) ConfigResolver {
	return func(cfg *conf.C) (*conf.C, error) {
		c := conf.NewConfig()
		if err := c.Merge(cfg); err != nil {
			return nil, err
		}
		return c, c.SetString("header", -1, header)
	}
}

func failResolve(*conf.C) (*conf.C, error) { return nil, errResolve }

// capturingPluginsReg returns a registry with a "capture" plugin recording
// the header setting it is created and updated with.
func capturingPluginsReg() (*plugin.PluginsReg, *[]string) {
	var headers []string
	capture := func(cfg *conf.C) error {
		var c struct {
			Header string `config:"header"`
		}
		err := cfg.Unpack(&c)
		headers = append(headers, c.Header)
		return err
	}
	reg := plugin.NewPluginsReg()
	_ = reg.Add(plugin.PluginFactory{
		Name: "capture",
		Make: func(_ string, cfg *conf.C, info beat.Info) (plugin.Plugin, error) {
			if err := capture(cfg); err != nil {
				return plugin.Plugin{Logger: info.Logger}, err
			}
			return plugin.Plugin{Jobs: createMockJob(), Endpoints: 1, Logger: info.Logger, DoUpdate: capture}, nil
		},
		Stats: plugin.NewPluginCountersRecorder("capture", monitoring.NewRegistry()),
	})
	return reg, &headers
}

func captureMonitorConf(t *testing.T, extra map[string]any) *conf.C {
	t.Helper()
	cfg, err := conf.NewConfigFrom(map[string]any{
		"type":     "capture",
		"id":       "secret-monitor",
		"name":     "secret monitor",
		"schedule": "@every 1m",
		"header":   unresolvedHeader,
	})
	require.NoError(t, err, "test config must be valid")
	if extra != nil {
		require.NoError(t, cfg.Merge(extra), "extra settings must merge")
	}
	return cfg
}

func testInfo(t *testing.T) beat.Info {
	return beat.Info{Logger: logptest.NewTestingLogger(t, "")}
}

func TestMonitorResolvesConfigOnlyForPlugin(t *testing.T) {
	reg, headers := capturingPluginsReg()
	cfg := captureMonitorConf(t, nil)

	m, err := newMonitor(cfg, reg, nil, nil, nil, testInfo(t), nil, resolveHeader("resolved"))
	require.NoError(t, err, "monitor must be created")
	defer m.Stop()
	assert.False(t, m.stdFields.BadConfig, "a resolved monitor must not be a bad config")

	require.NoError(t, m.Update(cfg), "update must resolve the config")
	assert.Equal(t, []string{"resolved", "resolved"}, *headers, "the plugin must get the resolved config on create and update")

	header, err := cfg.String("header", -1)
	require.NoError(t, err, "stored config must keep the header")
	assert.Equal(t, unresolvedHeader, header, "the stored config must not be modified")
}

func TestMonitorResolutionErrorReportsDown(t *testing.T) {
	reg, headers := capturingPluginsReg()

	pipel := &MockPipeline{}
	c, err := pipel.Connect()
	require.NoError(t, err, "pipeline must connect")
	sched := scheduler.Create(1, monitoring.NewRegistry(), time.Local, nil, false, logptest.NewTestingLogger(t, ""))
	defer sched.Stop()

	m, err := newMonitor(captureMonitorConf(t, nil), reg, c, sched.Add, nil, testInfo(t), nil, failResolve)
	require.NoError(t, err, "a resolution failure must not prevent creating the monitor")
	defer m.Stop()

	assert.True(t, m.stdFields.BadConfig, "the monitor must run the error job")
	assert.Empty(t, *headers, "the plugin must not be created")

	m.Start()
	require.Eventually(t, func() bool { return len(pipel.PublishedEvents()) > 0 }, 5*time.Second, 10*time.Millisecond, "an error event must be published")

	event := pipel.PublishedEvents()[0]
	monitorStatus, _ := event.Fields.GetValue("monitor.status")
	assert.Equal(t, "down", monitorStatus, "the monitor must be reported down")
	msg, _ := event.Fields.GetValue("error.message")
	assert.Contains(t, msg, errResolve.Error(), "the error event must explain the failure")
}

func TestMonitorUpdateFailsWhenConfigCannotBeResolved(t *testing.T) {
	reg, _ := capturingPluginsReg()
	m, err := newMonitor(captureMonitorConf(t, nil), reg, nil, nil, nil, testInfo(t), nil, resolveHeader("resolved"))
	require.NoError(t, err, "monitor must be created")
	defer m.Stop()

	m.resolveConfig = failResolve
	assert.ErrorIs(t, m.Update(captureMonitorConf(t, nil)), errResolve, "update must fail so the runner list recreates the monitor")
}

// The factory tests use an unknown store type: its error proves the factory
// goes through the secret stores resolver without registering a store.
var unknownSecretStore = map[string]any{secretstores.ConfigKey: []any{map[string]any{"type": "unknown"}}}

func TestFactoryCheckConfigValidatesSecretStores(t *testing.T) {
	reg, _ := capturingPluginsReg()
	factory, _, closeFactory := makeMockFactory(t, reg)
	defer closeFactory()

	err := factory.CheckConfig(captureMonitorConf(t, unknownSecretStore))
	assert.ErrorContains(t, err, `unknown secret store type "unknown"`, "CheckConfig must validate secret_stores")
}

func TestFactoryCreateResolvesSecretStores(t *testing.T) {
	reg, headers := capturingPluginsReg()
	factory, _, closeFactory := makeMockFactory(t, reg)
	defer closeFactory()

	runner, err := factory.Create(&MockPipeline{}, captureMonitorConf(t, unknownSecretStore))
	require.NoError(t, err, "a resolution failure must not prevent creating the monitor")
	defer runner.Stop()

	m, ok := runner.(*Monitor)
	require.True(t, ok, "the runner must be a monitor")
	assert.True(t, m.stdFields.BadConfig, "Create must resolve secret_stores")
	assert.Empty(t, *headers, "the plugin must not be created")
}
