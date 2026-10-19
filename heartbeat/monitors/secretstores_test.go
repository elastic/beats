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
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
)

const (
	testStoreType    = "monitors_test"
	unresolvedHeader = "ref://token"
)

var errResolve = errors.New("store unavailable")

// testStore replaces the header setting by its value, or fails. It's shared
// by all the monitors of this package's tests.
var testStore = &switchableStore{}

type switchableStore struct {
	mtx   sync.Mutex
	value string
	err   error
}

func (s *switchableStore) set(value string, err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.value, s.err = value, err
}

func (s *switchableStore) Validate(*conf.C, secretstores.Scope) error { return nil }

func (s *switchableStore) Replace(_ context.Context, cfg *conf.C, _ secretstores.Scope) (*conf.C, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	c := conf.NewConfig()
	if err := c.Merge(cfg); err != nil {
		return nil, err
	}
	return c, c.SetString("header", -1, s.value)
}

func init() {
	err := secretstores.Register(testStoreType, func(*conf.C, *logp.Logger) (secretstores.Store, error) {
		return testStore, nil
	})
	if err != nil {
		panic(err)
	}
}

// capturingPluginsReg returns a registry with a "capture" plugin, and a
// "browser" plugin for synthetics wrapping, recording the header setting
// they are created with.
func capturingPluginsReg() (*plugin.PluginsReg, func() []string) {
	var mtx sync.Mutex
	var headers []string
	reg := plugin.NewPluginsReg()
	for _, name := range []string{"capture", "browser"} {
		_ = reg.Add(plugin.PluginFactory{
			Name: name,
			Make: func(_ string, cfg *conf.C, info beat.Info) (plugin.Plugin, error) {
				var c struct {
					Header string `config:"header"`
				}
				if err := cfg.Unpack(&c); err != nil {
					return plugin.Plugin{}, err
				}
				mtx.Lock()
				headers = append(headers, c.Header)
				mtx.Unlock()
				return plugin.Plugin{Jobs: createMockJob(), Endpoints: 1, Logger: info.Logger}, nil
			},
			Stats: plugin.NewPluginCountersRecorder(name, monitoring.NewRegistry()),
		})
	}
	return reg, func() []string {
		mtx.Lock()
		defer mtx.Unlock()
		return append([]string(nil), headers...)
	}
}

func captureMonitorConf(t *testing.T, monitorType string, withSecretStores bool) *conf.C {
	t.Helper()
	m := map[string]any{
		"type":     monitorType,
		"id":       "secret-monitor",
		"name":     "secret monitor",
		"schedule": "@every 10ms",
		"header":   unresolvedHeader,
	}
	if withSecretStores {
		m[secretstores.ConfigKey] = []any{map[string]any{"type": testStoreType}}
	}
	cfg, err := conf.NewConfigFrom(m)
	require.NoError(t, err, "test config must be valid")
	return cfg
}

func TestFactoryResolvesOnlyWithSecretStores(t *testing.T) {
	testStore.set("resolved", nil)
	tests := map[string]struct {
		withSecretStores bool
		want             string
	}{
		"without secret_stores": {withSecretStores: false, want: unresolvedHeader},
		"with secret_stores":    {withSecretStores: true, want: "resolved"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			reg, headers := capturingPluginsReg()
			factory, _, closeFactory := makeMockFactory(t, reg)
			defer closeFactory()

			cfg := captureMonitorConf(t, "capture", tc.withSecretStores)
			runner, err := factory.Create(&MockPipeline{}, cfg)
			require.NoError(t, err, "monitor must be created")
			defer runner.Stop()

			assert.Equal(t, []string{tc.want}, headers(), "the plugin must get the expected header")
			header, err := cfg.String("header", -1)
			require.NoError(t, err, "stored config must keep the header")
			assert.Equal(t, unresolvedHeader, header, "the stored config must not be modified")
		})
	}
}

func TestFactoryCheckConfigValidatesSecretStores(t *testing.T) {
	reg, _ := capturingPluginsReg()
	factory, _, closeFactory := makeMockFactory(t, reg)
	defer closeFactory()

	cfg := captureMonitorConf(t, "capture", false)
	require.NoError(t, cfg.Merge(map[string]any{secretstores.ConfigKey: []any{map[string]any{"type": "unknown"}}}), "secret_stores must merge")
	err := factory.CheckConfig(cfg)
	assert.ErrorContains(t, err, `unknown secret store type "unknown"`, "CheckConfig must validate secret_stores")
}

// TestMonitorRecoversFromSecretStoreError runs monitors whose store fails,
// then recovers. Synthetics monitors use another wrapping, which must report
// the error too.
func TestMonitorRecoversFromSecretStoreError(t *testing.T) {
	for _, monitorType := range []string{"capture", "browser"} {
		t.Run(monitorType, func(t *testing.T) {
			testStore.set("", errResolve)
			reg, headers := capturingPluginsReg()

			pipel := &MockPipeline{}
			c, err := pipel.Connect()
			require.NoError(t, err, "pipeline must connect")
			logger := logptest.NewTestingLogger(t, "")
			sched := scheduler.Create(1, monitoring.NewRegistry(), time.Local, nil, false, logger)
			defer sched.Stop()

			resolver := secretstores.NewResolver(logger)
			m, err := newMonitor(captureMonitorConf(t, monitorType, true), reg, c, sched.Add, nil, beat.Info{Logger: logger}, nil, resolver.WrapFactory)
			require.NoError(t, err, "a store failure must not prevent creating the monitor")
			defer m.Stop()
			m.Start()

			field := func(e *beat.Event, key string) any {
				v, _ := e.Fields.GetValue(key)
				return v
			}
			var down *beat.Event
			require.Eventually(t, func() bool {
				for _, e := range pipel.PublishedEvents() {
					if field(e, "monitor.status") == "down" {
						down = e
						return true
					}
				}
				return false
			}, 5*time.Second, 5*time.Millisecond, "the monitor must report down while the store fails")
			assert.Contains(t, field(down, "error.message"), errResolve.Error(), "the down event must explain the failure")
			assert.Empty(t, headers(), "the plugin must not be created while the store fails")

			testStore.set("resolved", nil)
			require.Eventually(t, func() bool {
				events := pipel.PublishedEvents()
				return field(events[len(events)-1], "monitor.status") == "up"
			}, 5*time.Second, 5*time.Millisecond, "the monitor must report up once the store recovers")
			assert.Equal(t, "resolved", headers()[0], "the plugin must get the resolved header")
		})
	}
}
