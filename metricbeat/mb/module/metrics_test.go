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

package module

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beatmonitoring"
	"github.com/elastic/beats/v7/metricbeat/mb"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/paths"
)

type instrumentedFetcher struct {
	mb.BaseMetricSet
	started, release chan struct{}
	err              error
}

func (m *instrumentedFetcher) Fetch(r mb.ReporterV2) error {
	close(m.started)
	<-m.release
	if m.err == nil {
		r.Event(mb.Event{})
	}
	return m.err
}

func TestInstanceFetchMetrics(t *testing.T) {
	logger := logptest.NewTestingLogger(t, "")
	registry := mb.NewRegister()
	var fetchers []*instrumentedFetcher
	require.NoError(t, registry.AddMetricSet("instrumented", "fetcher", func(base mb.BaseMetricSet) (mb.MetricSet, error) {
		m := &instrumentedFetcher{BaseMetricSet: base, started: make(chan struct{}), release: make(chan struct{})}
		fetchers = append(fetchers, m)
		return m, nil
	}), "test metricset must register")
	config, err := conf.NewConfigFrom(map[string]any{"module": "instrumented", "metricsets": []string{"fetcher"}, "hosts": []string{"a", "b"}, "period": "1m"})
	require.NoError(t, err, "test config must unpack")
	mon := beatmonitoring.NewMonitoring()
	wrapper, err := NewWrapper(config, registry, &beat.Info{Logger: logger}, mon, paths.New())
	require.NoError(t, err, "test wrapper must build")
	fetchers[1].err = errors.New("collection failed")
	out := make(chan beat.Event, 2)
	done := make(chan struct{})
	first := wrapper.metricSets[0]
	reporter := &eventReporter{msw: first, out: out, done: done}
	finished := make(chan struct{})
	go func() { defer close(finished); first.fetch(context.Background(), reporter) }()
	select {
	case <-fetchers[0].started:
	case <-time.After(5 * time.Second):
		t.Fatal("fetch did not start")
	}
	assert.True(t, first.metrics.inProgress.Get(), "a blocked fetch must remain visible")
	assert.Positive(t, first.metrics.started.Get(), "the fetch start time must be available while blocked")
	close(fetchers[0].release)
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("fetch did not finish")
	}
	second := wrapper.metricSets[1]
	close(fetchers[1].release)
	second.fetch(context.Background(), &eventReporter{msw: second, out: out, done: done})
	assert.EqualValues(t, 1, first.metrics.fetches.Get(), "successful fetch must count once")
	assert.Zero(t, first.metrics.fetchErrors.Get(), "one instance's failure must not affect its sibling")
	assert.EqualValues(t, 1, first.metrics.events.Get(), "successful publication must count once")
	assert.EqualValues(t, 1, second.metrics.fetchErrors.Get(), "failed fetch must be recorded independently")
	assert.EqualValues(t, 1, second.metrics.consecutiveFailures.Get(), "failure streak must belong to the instance")
	assert.False(t, first.metrics.inProgress.Get(), "completed fetch must clear in-progress state")
	assert.Positive(t, first.metrics.duration.Count(), "completed fetch must have a duration observation")
	closed := make(chan struct{})
	close(closed)
	canceled := reporterV2{&eventReporter{msw: first, out: make(chan beat.Event), done: closed}}
	assert.False(t, canceled.Event(mb.Event{}), "a canceled unbuffered publication must stop")
	assert.EqualValues(t, 1, first.metrics.events.Get(), "a canceled handoff must not count as a publication")
	assert.EqualValues(t, 2, first.metrics.publishWait.Count(), "both successful and canceled handoffs must contribute wait observations")
	for _, ms := range wrapper.metricSets {
		releaseStats(mon.StatsRegistry(), ms.stats)
	}
}

type panickingFetcher struct{ mb.MetricSet }

func (panickingFetcher) Fetch(mb.ReporterV2) error { panic("fetch failed") }

func TestPanickingFetchMetrics(t *testing.T) {
	metrics := newInputMetrics(monitoring.NewRegistry(), logptest.NewTestingLogger(t, ""))
	wrapper := &metricSetWrapper{MetricSet: panickingFetcher{}, metrics: metrics}
	assert.Panics(t, func() { wrapper.fetch(context.Background(), &eventReporter{msw: wrapper}) }, "instrumentation must preserve panic recovery by the caller")
	assert.EqualValues(t, 1, metrics.fetchErrors.Get(), "a panicking fetch must not appear successful")
	assert.False(t, metrics.inProgress.Get(), "an unwound fetch must not remain in progress")
}
