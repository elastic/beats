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

//go:build !integration

package module_test

import (
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beatmonitoring"
	"github.com/elastic/beats/v7/libbeat/common/diagnostics"
	"github.com/elastic/beats/v7/libbeat/processors"
	pubtest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/module"
	_ "github.com/elastic/beats/v7/metricbeat/module/system"
	_ "github.com/elastic/beats/v7/metricbeat/module/system/cpu"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/paths"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunner(t *testing.T) {
	pubClient, factory := newPubClientFactory()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{reportingFetcherName},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create a new Wrapper based on the configuration.
	m, err := module.NewWrapper(config, mb.Registry, &beat.Info{Logger: logptest.NewTestingLogger(t, "")}, beatmonitoring.NewMonitoring(), paths.New(), module.WithMetricSetInfo())
	if err != nil {
		t.Fatal(err)
	}

	// Create the Runner facade.
	runner := module.NewRunner(factory(), m)

	// Start the module and have it publish to a new publisher.Client.
	runner.Start()

	assert.NotNil(t, <-pubClient.Channel)

	// Stop the module. This blocks until all MetricSets in the Module have
	// stopped and the publisher.Client is closed.
	runner.Stop()
}

func TestCPUDiagnostics(t *testing.T) {
	pubClient, factory := newPubClientFactory()

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"module":     "system",
		"metricsets": []string{"cpu"},
	})
	require.NoError(t, err)

	// Create a new Wrapper based on the configuration.
	m, err := module.NewWrapper(config, mb.Registry, &beat.Info{Logger: logptest.NewTestingLogger(t, "")}, beatmonitoring.NewMonitoring(), paths.New(), module.WithMetricSetInfo())
	if err != nil {
		t.Fatal(err)
	}

	runner := module.NewRunner(factory(), m)

	// First test, run before start. Shouldn't cause panics or other undefined behavior
	diag, ok := runner.(diagnostics.DiagnosticReporter)
	require.True(t, ok)
	diags := diag.Diagnostics()
	// This diagnostic set is only available on linux.
	// On other OSes, the list should be empty
	if runtime.GOOS == "linux" {
		require.NotEmpty(t, diags)
	} else {
		require.Empty(t, diags)
	}

	runner.Start()
	assert.NotNil(t, <-pubClient.Channel)

	diag, ok = runner.(diagnostics.DiagnosticReporter)
	require.True(t, ok)
	diags = diag.Diagnostics()
	if runtime.GOOS == "linux" {
		require.NotEmpty(t, diags)
		res := diags[0].Callback()
		require.NotEmpty(t, res)
	} else {
		require.Empty(t, diags)
	}

	runner.Stop()
	// stop, test again.
	diag, ok = runner.(diagnostics.DiagnosticReporter)
	require.True(t, ok)
	diags = diag.Diagnostics()
	if runtime.GOOS == "linux" {
		require.NotEmpty(t, diags)
	} else {
		require.Empty(t, diags)
	}

}

// newPubClientFactory returns a new ChanClient and a function that returns
// the same Client when invoked. This simulates the return value of
// Publisher.Connect.
func newPubClientFactory() (*pubtest.ChanClient, func() beat.Client) {
	client := pubtest.NewChanClient(10)
	return client, func() beat.Client { return client }
}

func TestRunnerStop_ClientClosedAfterPublishGoroutine(t *testing.T) {
	// Wrap a no-op processor with SafeWrap so it becomes a safeProcessorWithClose.
	proc, err := processors.SafeWrap(func(_ *conf.C, _ *logp.Logger) (beat.Processor, error) {
		return noopCloser{}, nil
	})(nil, nil)
	require.NoError(t, err)

	publishing := make(chan struct{})
	unblock := make(chan struct{})

	var runErr atomic.Pointer[error]

	client := &blockingClient{
		onPublish: func(e beat.Event) {
			close(publishing)
			<-unblock
			// This is the call that fails in production when the processor
			// chain is closed before the publish goroutine finishes.
			if _, err := proc.Run(&e); err != nil {
				runErr.Store(&err)
			}
		},
		onClose: func() error {
			return processors.Close(proc)
		},
	}

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{pushMetricSetName},
		"period":     "1s",
	})
	require.NoError(t, err)

	m, err := module.NewWrapper(config, mb.Registry, &beat.Info{Logger: logptest.NewTestingLogger(t, "")}, beatmonitoring.NewMonitoring(), paths.New(), module.WithMetricSetInfo())
	require.NoError(t, err)

	r := module.NewRunner(client, m)
	r.Start()
	<-publishing // a publish is now in progress and blocked

	// r.Stop() waits for in-flight publish goroutines before calling
	// client.Close(), so we must release the blocked goroutine from here.
	go close(unblock)
	r.Stop()

	if p := runErr.Load(); p != nil {
		assert.NoError(t, *p, "proc.Run failed inside Publish — processor was closed before the publish goroutine finished")
	}
}

// noopCloser is a pass-through processor that implements processors.Closer.
// The Closer interface is what causes SafeWrap to produce a safeProcessorWithClose,
// which tracks whether Close has been called and returns ErrClosed from Run afterwards.
type noopCloser struct{}

func (noopCloser) Run(e *beat.Event) (*beat.Event, error) { return e, nil }
func (noopCloser) Close() error                           { return nil }
func (noopCloser) String() string                         { return "noop" }

type blockingClient struct {
	onPublish func(beat.Event)
	onClose   func() error
}

func (c *blockingClient) Publish(e beat.Event) { c.onPublish(e) }
func (c *blockingClient) PublishAll(es []beat.Event) {
	for _, e := range es {
		c.onPublish(e)
	}
}
func (c *blockingClient) Close() error { return c.onClose() }
