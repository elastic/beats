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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beatmonitoring"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/module"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/paths"
)

const (
	multiFetcherName = "MultiFetcher"
)

func init() {
	mb.Registry.MustAddMetricSet(moduleName, multiFetcherName, newFakeMultiFetcher)
}

// fakeMultiFetcher emits multiple events per fetch cycle.
type fakeMultiFetcher struct {
	mb.BaseMetricSet
}

func (ms *fakeMultiFetcher) Fetch(r mb.ReporterV2) {
	for i := 0; i < 3; i++ {
		r.Event(mb.Event{MetricSetFields: map[string]interface{}{"value": i}})
	}
}

func newFakeMultiFetcher(base mb.BaseMetricSet) (mb.MetricSet, error) {
	var r mb.ReportingMetricSetV2 = &fakeMultiFetcher{BaseMetricSet: base}
	return r, nil
}

// trackingClient records Publish vs PublishAll calls.
type trackingClient struct {
	mu              sync.Mutex
	publishCalls    int
	publishAllCalls int
	events          []beat.Event
	closed          bool
}

func (c *trackingClient) Publish(event beat.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.publishCalls++
	c.events = append(c.events, event)
}

func (c *trackingClient) PublishAll(events []beat.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.publishAllCalls++
	c.events = append(c.events, events...)
}

func (c *trackingClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *trackingClient) getStats() (publishCalls, publishAllCalls, totalEvents int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.publishCalls, c.publishAllCalls, len(c.events)
}

func TestBatchedRunnerPublishesViaPublishAll(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	registry := newTestRegistry(t)
	err := registry.AddMetricSet(moduleName, multiFetcherName, newFakeMultiFetcher)
	require.NoError(t, err)

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{reportingFetcherName},
		"period":     "100ms",
	})
	require.NoError(t, err)

	factory := module.NewBatchedFactory(
		beat.Info{Logger: logger, Paths: paths.New()},
		beatmonitoring.NewMonitoring(),
		registry,
		module.WithMetricSetInfo(),
		module.WithMaxStartDelay(0),
	)

	client := &trackingClient{}
	pipeline := &fakePipeline{client: client}

	runner, err := factory.Create(pipeline, config)
	require.NoError(t, err)

	runner.Start()

	// Wait for at least one fetch cycle to complete.
	require.Eventually(t, func() bool {
		_, _, total := client.getStats()
		return total >= 1
	}, 5*time.Second, 10*time.Millisecond, "expected at least one event")

	runner.Stop()

	pubCalls, pubAllCalls, totalEvents := client.getStats()
	assert.Equal(t, 0, pubCalls, "batched runner should not call Publish")
	assert.Greater(t, pubAllCalls, 0, "batched runner should call PublishAll")
	assert.Greater(t, totalEvents, 0, "should have received events")
}

func TestBatchedRunnerMultiEventMetricSet(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	registry := newTestRegistry(t)
	err := registry.AddMetricSet(moduleName, multiFetcherName, newFakeMultiFetcher)
	require.NoError(t, err)

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{multiFetcherName},
		"period":     "100ms",
	})
	require.NoError(t, err)

	factory := module.NewBatchedFactory(
		beat.Info{Logger: logger, Paths: paths.New()},
		beatmonitoring.NewMonitoring(),
		registry,
		module.WithMaxStartDelay(0),
	)

	client := &trackingClient{}
	pipeline := &fakePipeline{client: client}

	runner, err := factory.Create(pipeline, config)
	require.NoError(t, err)

	runner.Start()

	// MultiFetcher emits 3 events per fetch. Wait for at least one cycle.
	require.Eventually(t, func() bool {
		_, _, total := client.getStats()
		return total >= 3
	}, 5*time.Second, 10*time.Millisecond, "expected at least 3 events")

	runner.Stop()

	pubCalls, pubAllCalls, totalEvents := client.getStats()
	assert.Equal(t, 0, pubCalls, "batched runner should not call Publish")
	// Each fetch cycle should produce exactly one PublishAll call with 3 events.
	assert.Greater(t, pubAllCalls, 0, "should have at least one PublishAll call")
	assert.Equal(t, 0, totalEvents%3, "events should come in multiples of 3")
}

func TestBatchedRunnerMultipleMetricSets(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	registry := newTestRegistry(t)
	err := registry.AddMetricSet(moduleName, multiFetcherName, newFakeMultiFetcher)
	require.NoError(t, err)

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{reportingFetcherName, multiFetcherName},
		"period":     "100ms",
	})
	require.NoError(t, err)

	factory := module.NewBatchedFactory(
		beat.Info{Logger: logger, Paths: paths.New()},
		beatmonitoring.NewMonitoring(),
		registry,
		module.WithMaxStartDelay(0),
	)

	// Track per-client stats. The factory creates one client per metricset.
	var clients []*trackingClient
	var mu sync.Mutex
	pipeline := &fakePipelineFunc{connectFn: func() (beat.Client, error) {
		c := &trackingClient{}
		mu.Lock()
		clients = append(clients, c)
		mu.Unlock()
		return c, nil
	}}

	runner, err := factory.Create(pipeline, config)
	require.NoError(t, err)

	runner.Start()

	// Wait for events from both metricsets.
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		if len(clients) < 2 {
			return false
		}
		_, _, t1 := clients[0].getStats()
		_, _, t2 := clients[1].getStats()
		return t1 >= 1 && t2 >= 1
	}, 5*time.Second, 10*time.Millisecond, "expected events from both metricsets")

	runner.Stop()

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, clients, 2, "should have two clients (one per metricset)")
	for _, c := range clients {
		pub, pubAll, _ := c.getStats()
		assert.Equal(t, 0, pub, "batched runner should not call Publish")
		assert.Greater(t, pubAll, 0, "each metricset should use PublishAll")
	}
}

func TestBatchedRunnerPushMetricSetFallsBackToStandardRunner(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	registry := newTestRegistry(t)

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{pushMetricSetName},
		"period":     "100ms",
	})
	require.NoError(t, err)

	factory := module.NewBatchedFactory(
		beat.Info{Logger: logger, Paths: paths.New()},
		beatmonitoring.NewMonitoring(),
		registry,
		module.WithMaxStartDelay(0),
	)

	client := &trackingClient{}
	pipeline := &fakePipeline{client: client}

	runner, err := factory.Create(pipeline, config)
	require.NoError(t, err)

	runner.Start()

	// Push metricset should publish one event then wait for done.
	require.Eventually(t, func() bool {
		_, _, total := client.getStats()
		return total >= 1
	}, 5*time.Second, 10*time.Millisecond, "expected at least one event from push metricset")

	runner.Stop()

	// Push metricsets use the standard runner which calls Publish, not PublishAll.
	pubCalls, _, _ := client.getStats()
	assert.Greater(t, pubCalls, 0, "push metricset should use Publish (standard runner)")
}

func TestBatchedRunnerStopIsClean(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	registry := newTestRegistry(t)

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{reportingFetcherName},
		"period":     "10s", // long period so we only get the initial fetch
	})
	require.NoError(t, err)

	factory := module.NewBatchedFactory(
		beat.Info{Logger: logger, Paths: paths.New()},
		beatmonitoring.NewMonitoring(),
		registry,
		module.WithMaxStartDelay(0),
	)

	client := &trackingClient{}
	pipeline := &fakePipeline{client: client}

	runner, err := factory.Create(pipeline, config)
	require.NoError(t, err)

	runner.Start()

	// Wait for initial fetch.
	require.Eventually(t, func() bool {
		_, _, total := client.getStats()
		return total >= 1
	}, 5*time.Second, 10*time.Millisecond)

	// Stop should not hang or panic.
	stopped := make(chan struct{})
	go func() {
		runner.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		// Clean shutdown.
	case <-time.After(5 * time.Second):
		t.Fatal("Stop() did not return within 5 seconds")
	}

	client.mu.Lock()
	assert.True(t, client.closed, "client should be closed after Stop")
	client.mu.Unlock()
}

func TestNonBatchedFactoryUsesPublish(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	registry := newTestRegistry(t)

	config, err := conf.NewConfigFrom(map[string]interface{}{
		"module":     moduleName,
		"metricsets": []string{reportingFetcherName},
		"period":     "100ms",
	})
	require.NoError(t, err)

	// Use the standard (non-batched) factory.
	factory := module.NewFactory(
		beat.Info{Logger: logger, Paths: paths.New()},
		beatmonitoring.NewMonitoring(),
		registry,
		module.WithMetricSetInfo(),
		module.WithMaxStartDelay(0),
	)

	var publishCount atomic.Int64
	var publishAllCount atomic.Int64
	pipeline := &fakePipelineFunc{connectFn: func() (beat.Client, error) {
		return &countingClient{
			publishCount:    &publishCount,
			publishAllCount: &publishAllCount,
		}, nil
	}}

	runner, err := factory.Create(pipeline, config)
	require.NoError(t, err)

	runner.Start()

	require.Eventually(t, func() bool {
		return publishCount.Load() >= 1
	}, 5*time.Second, 10*time.Millisecond)

	runner.Stop()

	assert.Greater(t, publishCount.Load(), int64(0), "standard factory should use Publish")
	assert.Equal(t, int64(0), publishAllCount.Load(), "standard factory should not use PublishAll")
}

// --- test helpers ---

// fakePipeline returns the same client for every ConnectWith call.
type fakePipeline struct {
	client beat.Client
}

func (p *fakePipeline) Connect() (beat.Client, error) {
	return p.client, nil
}

func (p *fakePipeline) ConnectWith(beat.ClientConfig) (beat.Client, error) {
	return p.client, nil
}

// fakePipelineFunc calls a function for each ConnectWith.
type fakePipelineFunc struct {
	connectFn func() (beat.Client, error)
}

func (p *fakePipelineFunc) Connect() (beat.Client, error) {
	return p.connectFn()
}

func (p *fakePipelineFunc) ConnectWith(beat.ClientConfig) (beat.Client, error) {
	return p.connectFn()
}

// countingClient counts Publish vs PublishAll calls via atomics.
type countingClient struct {
	publishCount    *atomic.Int64
	publishAllCount *atomic.Int64
}

func (c *countingClient) Publish(event beat.Event) {
	c.publishCount.Add(1)
}

func (c *countingClient) PublishAll(events []beat.Event) {
	c.publishAllCount.Add(1)
}

func (c *countingClient) Close() error { return nil }
