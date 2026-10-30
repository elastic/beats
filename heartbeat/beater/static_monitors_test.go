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

package beater

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type fakeRunner struct{ started, stopped bool }

func (r *fakeRunner) Start()         { r.started = true }
func (r *fakeRunner) Stop()          { r.stopped = true }
func (r *fakeRunner) String() string { return "fakeRunner" }

// fakeFactory fails to create the monitor whose id is badID.
type fakeFactory struct {
	badID   string
	runners map[string]*fakeRunner
}

func (f *fakeFactory) Create(_ beat.PipelineConnector, c *conf.C) (cfgfile.Runner, error) {
	id, err := c.String("id", -1)
	if err != nil {
		return nil, err
	}
	if id == f.badID {
		return nil, errors.New("invalid monitor config")
	}
	r := &fakeRunner{}
	f.runners[id] = r
	return r, nil
}

func (f *fakeFactory) CheckConfig(*conf.C) error { return nil }

func newStaticMonitorsHeartbeat(t *testing.T, otel bool) (*Heartbeat, *fakeFactory) {
	t.Helper()
	cfg := &config.Config{}
	for _, id := range []string{"a", "bad", "c"} {
		c, err := conf.NewConfigFrom(map[string]any{"id": id})
		require.NoError(t, err, "could not build monitor config")
		cfg.Monitors = append(cfg.Monitors, c)
	}
	factory := &fakeFactory{badID: "bad", runners: map[string]*fakeRunner{}}
	logger := logptest.NewTestingLogger(t, "")
	bt := &Heartbeat{config: cfg, monitorFactory: factory, logger: logger}
	if otel {
		// As set up by Run in receiver mode.
		bt.monitorFactory = invalidMonitorFactory{RunnerFactory: factory, logger: logger}
	}
	return bt, factory
}

func TestRunStaticMonitorsInvalidMonitor(t *testing.T) {
	t.Run("receiver mode starts the valid monitors", func(t *testing.T) {
		bt, factory := newStaticMonitorsHeartbeat(t, true)

		stop, err := bt.RunStaticMonitors(nil, nil)
		require.NoError(t, err, "an invalid monitor must not fail the receiver")
		require.Len(t, factory.runners, 2, "the monitors around the invalid one must be created")
		for id, r := range factory.runners {
			assert.True(t, r.started, "monitor %s must be started", id)
		}

		stop()
		for id, r := range factory.runners {
			assert.True(t, r.stopped, "monitor %s must be stopped by stop()", id)
		}
	})

	t.Run("standalone mode still fails fast", func(t *testing.T) {
		bt, _ := newStaticMonitorsHeartbeat(t, false)

		_, err := bt.RunStaticMonitors(nil, nil)
		require.Error(t, err, "standalone Heartbeat must keep rejecting an invalid monitor")
	})
}
