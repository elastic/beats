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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/management/status"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type statusUpdate struct {
	state status.Status
	msg   string
}

type recordingReporter struct {
	mtx     sync.Mutex
	updates []statusUpdate
}

func (r *recordingReporter) UpdateStatus(s status.Status, msg string) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	r.updates = append(r.updates, statusUpdate{s, msg})
}

type erroringFactory struct{ err error }

func (f erroringFactory) Create(beat.PipelineConnector, *conf.C) (cfgfile.Runner, error) {
	return nil, f.err
}

func (erroringFactory) CheckConfig(*conf.C) error { return nil }

func TestInvalidMonitorFactory(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")

	t.Run("an invalid monitor reports its error while started", func(t *testing.T) {
		factory := invalidMonitorFactory{RunnerFactory: erroringFactory{err: errors.New("missing schedule")}, logger: logger}

		runner, err := factory.Create(nil, conf.NewConfig())
		require.NoError(t, err, "an invalid monitor must not fail the receiver")
		withStatus, ok := runner.(status.WithStatusReporter)
		require.True(t, ok, "the placeholder must accept a status reporter")
		reporter := &recordingReporter{}
		withStatus.SetStatusReporter(reporter)

		runner.Start()
		runner.Stop()

		assert.Equal(t, []statusUpdate{
			{status.Failed, "missing schedule"},
			{status.Stopped, ""},
		}, reporter.updates, "the failure must be reported while started and cleared once stopped")
	})

	t.Run("a disabled monitor is still skipped", func(t *testing.T) {
		factory := invalidMonitorFactory{RunnerFactory: erroringFactory{err: monitors.ErrMonitorDisabled}, logger: logger}

		_, err := factory.Create(nil, conf.NewConfig())
		assert.ErrorIs(t, err, monitors.ErrMonitorDisabled, "disabled monitors must keep their error")
	})
}
