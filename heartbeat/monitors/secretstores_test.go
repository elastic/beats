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
	"maps"
	"sync"
	"sync/atomic"
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
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/heartbeat/scheduler"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
)

const testSecretStore = "monsecret"

var testStoreCalls atomic.Int64

type testSecretStoreImpl struct{ fail bool }

func (s testSecretStoreImpl) Resolve(_ context.Context, path, field string) (string, error) {
	testStoreCalls.Add(1)
	if s.fail {
		return "", errors.New("monsecret: store unavailable")
	}
	return path + "-" + field + "-value", nil
}

func init() {
	secretstores.Register(testSecretStore, func(cfg *conf.C, _ *logp.Logger) (secretstores.Store, error) {
		var c struct {
			Fail bool `config:"fail"`
		}
		if err := cfg.Unpack(&c); err != nil {
			return nil, err
		}
		return testSecretStoreImpl{fail: c.Fail}, nil
	})
}

// capturingPluginsReg returns a registry with a "capture" plugin recording
// the token setting it is created and updated with.
func capturingPluginsReg() (*plugin.PluginsReg, *capturedTokens) {
	captured := &capturedTokens{}
	reg := plugin.NewPluginsReg()
	unpackToken := func(cfg *conf.C) (string, error) {
		var c struct {
			Header string `config:"header"`
		}
		err := cfg.Unpack(&c)
		return c.Header, err
	}
	_ = reg.Add(plugin.PluginFactory{
		Name: "capture",
		Make: func(_ string, cfg *conf.C, info beat.Info) (plugin.Plugin, error) {
			token, err := unpackToken(cfg)
			if err != nil {
				return plugin.Plugin{Logger: info.Logger}, err
			}
			captured.add(token)
			return plugin.Plugin{
				Jobs:      createMockJob(),
				Endpoints: 1,
				Logger:    info.Logger,
				DoUpdate: func(cfg *conf.C) error {
					token, err := unpackToken(cfg)
					captured.add(token)
					return err
				},
			}, nil
		},
		Stats: plugin.NewPluginCountersRecorder("capture", monitoring.NewRegistry()),
	})
	return reg, captured
}

type capturedTokens struct {
	mu     sync.Mutex
	tokens []string
}

func (c *capturedTokens) add(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokens = append(c.tokens, token)
}

func (c *capturedTokens) get() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.tokens...)
}

func secretMonitorConf(t *testing.T, storeEntry map[string]any) *conf.C {
	t.Helper()
	entry := map[string]any{"type": testSecretStore}
	maps.Copy(entry, storeEntry)
	cfg, err := conf.NewConfigFrom(map[string]any{
		"type":                     "capture",
		"id":                       "secret-monitor",
		"name":                     "secret monitor",
		"schedule":                 "@every 1m",
		"header":                   "Bearer $monsecret{app/creds#token}",
		secretstores.ConfigKey:     []any{entry},
		"fields.unrelated_setting": "kept",
	})
	require.NoError(t, err, "test config must be valid")
	return cfg
}

func TestMonitorResolvesSecretsOnlyForPlugin(t *testing.T) {
	reg, captured := capturingPluginsReg()
	cfg := secretMonitorConf(t, nil)
	testStoreCalls.Store(0)

	// Generic config handling, with the default process-wide options, keeps
	// the reference as text and never calls the store.
	hashBefore, err := cfgfile.HashConfig(cfg)
	require.NoError(t, err, "a config with secret references must be hashable")
	_, err = stdfields.ConfigToStdMonitorFields(cfg)
	require.NoError(t, err, "monitor std fields must be readable")
	assert.Zero(t, testStoreCalls.Load(), "hashing must not call the secret store")

	resolver := secretstores.NewResolver(logptest.NewTestingLogger(t, ""))
	m, err := newMonitor(cfg, reg, nil, nil, nil, beat.Info{Logger: logptest.NewTestingLogger(t, "")}, nil, resolver.Resolve)
	require.NoError(t, err, "monitor must be created")
	defer m.Stop()

	assert.Equal(t, []string{"Bearer app/creds-token-value"}, captured.get(), "the plugin must get the resolved value")
	assert.False(t, m.stdFields.BadConfig, "a resolved monitor must not be a bad config")

	hashAfter, err := cfgfile.HashConfig(cfg)
	require.NoError(t, err, "the config must still be hashable")
	assert.Equal(t, hashBefore, hashAfter, "the stored config must not be modified")

	var stored struct {
		Header string `config:"header"`
	}
	require.NoError(t, cfg.Unpack(&stored), "stored config must unpack")
	assert.Equal(t, "Bearer $monsecret{app/creds#token}", stored.Header, "the stored config must keep the reference")

	require.NoError(t, m.Update(cfg), "update must resolve references")
	assert.Equal(t, []string{"Bearer app/creds-token-value", "Bearer app/creds-token-value"}, captured.get(), "the plugin must get resolved values on update")
}

func TestMonitorWithSecretsGetsAutoID(t *testing.T) {
	reg, _ := capturingPluginsReg()
	cfg := secretMonitorConf(t, nil)
	_, err := cfg.Remove("id", -1)
	require.NoError(t, err, "id must be removable")

	resolver := secretstores.NewResolver(logptest.NewTestingLogger(t, ""))
	m, err := newMonitor(cfg, reg, nil, nil, nil, beat.Info{Logger: logptest.NewTestingLogger(t, "")}, nil, resolver.Resolve)
	require.NoError(t, err, "a monitor without id must be created from the unresolved config")
	defer m.Stop()
	assert.Contains(t, m.stdFields.ID, "auto-capture-", "an id must be generated")
}

func TestMonitorSecretResolutionErrorReportsDown(t *testing.T) {
	reg, captured := capturingPluginsReg()
	cfg := secretMonitorConf(t, map[string]any{"fail": true})

	pipel := &MockPipeline{}
	c, err := pipel.Connect()
	require.NoError(t, err, "pipeline must connect")

	sched := scheduler.Create(1, monitoring.NewRegistry(), time.Local, nil, false, logptest.NewTestingLogger(t, ""))
	defer sched.Stop()

	resolver := secretstores.NewResolver(logptest.NewTestingLogger(t, ""))
	m, err := newMonitor(cfg, reg, c, sched.Add, nil, beat.Info{Logger: logptest.NewTestingLogger(t, "")}, nil, resolver.Resolve)
	require.NoError(t, err, "a secret resolution failure must not prevent creating the monitor")
	defer m.Stop()

	assert.True(t, m.stdFields.BadConfig, "the monitor must run the error job")
	assert.Empty(t, captured.get(), "the plugin must not be created")

	m.Start()
	require.Eventually(t, func() bool { return len(pipel.PublishedEvents()) > 0 }, 5*time.Second, 10*time.Millisecond, "an error event must be published")

	event := pipel.PublishedEvents()[0]
	monitorStatus, _ := event.Fields.GetValue("monitor.status")
	assert.Equal(t, "down", monitorStatus, "the monitor must be reported down")
	msg, _ := event.Fields.GetValue("error.message")
	assert.Contains(t, msg, "monsecret: store unavailable", "the error event must explain the failure")
}

func TestMonitorUpdateFailsWhenSecretsCannotBeResolved(t *testing.T) {
	reg, _ := capturingPluginsReg()

	resolver := secretstores.NewResolver(logptest.NewTestingLogger(t, ""))
	m, err := newMonitor(secretMonitorConf(t, nil), reg, nil, nil, nil, beat.Info{Logger: logptest.NewTestingLogger(t, "")}, nil, resolver.Resolve)
	require.NoError(t, err, "monitor must be created")
	defer m.Stop()

	err = m.Update(secretMonitorConf(t, map[string]any{"fail": true}))
	assert.ErrorContains(t, err, "monsecret: store unavailable", "update must fail so the runner list recreates the monitor")
}

func TestFactoryCheckConfigValidatesSecretStores(t *testing.T) {
	reg, _ := capturingPluginsReg()
	factory, _, closeFactory := makeMockFactory(t, reg)
	defer closeFactory()
	testStoreCalls.Store(0)

	cfg, err := conf.NewConfigFrom(map[string]any{
		"type":     "http",
		"id":       "no-store",
		"schedule": "@every 1m",
		"header":   "$monsecret{app/creds#token}",
	})
	require.NoError(t, err, "test config must be valid")

	assert.ErrorContains(t, factory.CheckConfig(cfg), `requires a "monsecret" entry`, "a reference without a store must be rejected")
	assert.Zero(t, testStoreCalls.Load(), "checking a config must not call the secret store")
}
